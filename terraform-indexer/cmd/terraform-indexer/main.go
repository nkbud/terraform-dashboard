package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/collector"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/db"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/parser"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/queue"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/utils"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/writer"
)

var (
	configFile string
	logger     *utils.Logger
)

var rootCmd = &cobra.Command{
	Use:   "terraform-indexer",
	Short: "A scalable Terraform file ingestion and indexing service",
	Long: `terraform-indexer is a service that polls .tfstate and .tf files from external sources
(S3, Kubernetes Secrets, Bitbucket), parses them into Terraform language constructs,
and stores the data in a searchable PostgreSQL database.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the indexer server",
	Run:   runIndexer,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.yaml", "config file")
	rootCmd.AddCommand(serverCmd)
	
	logger = utils.NewLogger()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runIndexer(cmd *cobra.Command, args []string) {
	// Load configuration
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatalf("Failed to read config file: %v", err)
	}
	
	// Set log level
	logLevel := viper.GetString("logging.level")
	logger.SetLevel(logLevel)
	
	logger.Info("Starting terraform-indexer")
	
	// Initialize metrics
	metrics := utils.NewMetrics()
	
	// Initialize database
	dbConfig := db.Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Database: viper.GetString("database.database"),
		SSLMode:  viper.GetString("database.ssl_mode"),
	}
	
	database, err := db.NewDB(dbConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	
	// Run migrations
	if err := database.Migrate(); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}
	
	logger.Info("Database connected and migrated")
	
	// Initialize components
	fileQueue := queue.NewFileQueue()
	objectQueue := queue.NewObjectQueue()
	
	parserRegistry := parser.NewParserRegistry()
	parserRegistry.Register(parser.NewStateParser())
	parserRegistry.Register(parser.NewTerraformParser())
	
	dbWriter := writer.NewDatabaseWriter(database)
	
	// Initialize collectors
	var collectors []collector.Collector
	for _, collectorConfig := range viper.Get("collectors").([]interface{}) {
		config := collectorConfig.(map[string]interface{})
		if enabled, ok := config["enabled"].(bool); !ok || !enabled {
			continue
		}
		
		name := config["name"].(string)
		settings := config["settings"].(map[string]interface{})
		sourceStr := settings["source"].(string)
		source := model.FileSource(sourceStr)
		
		coll := collector.NewMockCollector(name, source)
		collectors = append(collectors, coll)
		logger.Infof("Initialized collector: %s (%s)", name, source)
	}
	
	// Start HTTP server for metrics and health checks
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		healthHandler(w, r, database)
	})
	if viper.GetBool("metrics.enabled") {
		http.Handle(viper.GetString("metrics.path"), promhttp.Handler())
	}
	
	serverPort := viper.GetInt("server.port")
	serverHost := viper.GetString("server.host")
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", serverHost, serverPort),
	}
	
	go func() {
		logger.Infof("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()
	
	// Start worker goroutines
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	
	// Collector workers
	for i, coll := range collectors {
		wg.Add(1)
		go collectorWorker(ctx, &wg, coll, fileQueue, metrics, fmt.Sprintf("collector-%d", i))
	}
	
	// Parser workers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go parserWorker(ctx, &wg, parserRegistry, fileQueue, objectQueue, metrics, fmt.Sprintf("parser-%d", i))
	}
	
	// Writer workers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go writerWorker(ctx, &wg, dbWriter, objectQueue, metrics, fmt.Sprintf("writer-%d", i))
	}
	
	// Queue metrics updater
	wg.Add(1)
	go queueMetricsWorker(ctx, &wg, fileQueue, objectQueue, metrics)
	
	logger.Info("All workers started")
	
	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	logger.Info("Shutting down...")
	
	// Cancel context and wait for workers
	cancel()
	
	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
	
	wg.Wait()
	logger.Info("Shutdown complete")
}

func collectorWorker(ctx context.Context, wg *sync.WaitGroup, coll collector.Collector, fileQueue queue.FileQueue, metrics *utils.Metrics, workerID string) {
	defer wg.Done()
	
	interval := viper.GetDuration("polling.interval")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	logger.Infof("Starting collector worker: %s", coll.Name())
	metrics.WorkerStatus.WithLabelValues("collector", workerID).Set(1)
	defer metrics.WorkerStatus.WithLabelValues("collector", workerID).Set(0)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			timer := prometheus.NewTimer(metrics.ProcessingTime.WithLabelValues("collector", "collect"))
			files, err := coll.Collect(ctx)
			timer.ObserveDuration()
			
			if err != nil {
				logger.Errorf("Collector %s failed: %v", coll.Name(), err)
				metrics.ProcessingErrors.WithLabelValues("collector", "collect_failed").Inc()
				continue
			}
			
			for _, file := range files {
				if err := fileQueue.Enqueue(ctx, file); err != nil {
					logger.Errorf("Failed to enqueue file: %v", err)
					metrics.ProcessingErrors.WithLabelValues("collector", "enqueue_failed").Inc()
				} else {
					metrics.FilesCollected.WithLabelValues(string(file.Source), string(file.FileType)).Inc()
				}
			}
			
			logger.Debugf("Collector %s collected %d files", coll.Name(), len(files))
		}
	}
}

func parserWorker(ctx context.Context, wg *sync.WaitGroup, parserRegistry *parser.ParserRegistry, fileQueue queue.FileQueue, objectQueue queue.ObjectQueue, metrics *utils.Metrics, workerID string) {
	defer wg.Done()
	
	logger.Info("Starting parser worker")
	metrics.WorkerStatus.WithLabelValues("parser", workerID).Set(1)
	defer metrics.WorkerStatus.WithLabelValues("parser", workerID).Set(0)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			file, err := fileQueue.Dequeue(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Errorf("Failed to dequeue file: %v", err)
				metrics.ProcessingErrors.WithLabelValues("parser", "dequeue_failed").Inc()
				continue
			}
			
			timer := prometheus.NewTimer(metrics.ProcessingTime.WithLabelValues("parser", "parse"))
			objects, err := parserRegistry.Parse(ctx, file)
			timer.ObserveDuration()
			
			if err != nil {
				logger.Errorf("Failed to parse file %s: %v", file.ID, err)
				metrics.ProcessingErrors.WithLabelValues("parser", "parse_failed").Inc()
				continue
			}
			
			for _, obj := range objects {
				if err := objectQueue.Enqueue(ctx, obj); err != nil {
					logger.Errorf("Failed to enqueue object: %v", err)
					metrics.ProcessingErrors.WithLabelValues("parser", "enqueue_failed").Inc()
				} else {
					metrics.ObjectsParsed.WithLabelValues(string(obj.Type), string(file.FileType)).Inc()
				}
			}
			
			logger.Debugf("Parsed file %s into %d objects", file.ID, len(objects))
		}
	}
}

func writerWorker(ctx context.Context, wg *sync.WaitGroup, dbWriter writer.Writer, objectQueue queue.ObjectQueue, metrics *utils.Metrics, workerID string) {
	defer wg.Done()
	
	logger.Info("Starting writer worker")
	metrics.WorkerStatus.WithLabelValues("writer", workerID).Set(1)
	defer metrics.WorkerStatus.WithLabelValues("writer", workerID).Set(0)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			obj, err := objectQueue.Dequeue(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Errorf("Failed to dequeue object: %v", err)
				metrics.ProcessingErrors.WithLabelValues("writer", "dequeue_failed").Inc()
				continue
			}
			
			timer := prometheus.NewTimer(metrics.ProcessingTime.WithLabelValues("writer", "write"))
			if err := dbWriter.WriteObject(ctx, obj); err != nil {
				logger.Errorf("Failed to write object %s: %v", obj.ID, err)
				metrics.ProcessingErrors.WithLabelValues("writer", "write_failed").Inc()
				timer.ObserveDuration()
				continue
			}
			timer.ObserveDuration()
			
			metrics.ObjectsWritten.WithLabelValues(string(obj.Type)).Inc()
			logger.Debugf("Wrote object %s to database", obj.ID)
		}
	}
}

func queueMetricsWorker(ctx context.Context, wg *sync.WaitGroup, fileQueue queue.FileQueue, objectQueue queue.ObjectQueue, metrics *utils.Metrics) {
	defer wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.QueueSize.WithLabelValues("files").Set(float64(fileQueue.Size()))
			metrics.QueueSize.WithLabelValues("objects").Set(float64(objectQueue.Size()))
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request, database *db.DB) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check database connectivity
	if err := database.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unhealthy","error":"database_connection_failed"}`))
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}