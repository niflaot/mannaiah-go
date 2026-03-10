package runtime

// Config defines assets integration configuration values.
type Config struct {
	// JPGWorkerEnabled enables scheduled JPG worker behavior.
	JPGWorkerEnabled bool `mapstructure:"ASSETS_JPG_WORKER_ENABLED" default:"false"`
	// JPGWorkerCron defines cron specs for scheduled JPG worker behavior.
	JPGWorkerCron string `mapstructure:"ASSETS_JPG_WORKER_CRON" default:"0 * * * *"`
	// JPGWorkerTags defines comma-separated asset tag names eligible for JPG conversion.
	JPGWorkerTags string `mapstructure:"ASSETS_JPG_WORKER_TAGS" default:""`
	// JPGWorkerBatchSize defines asset page sizes processed per worker execution.
	JPGWorkerBatchSize int `mapstructure:"ASSETS_JPG_WORKER_BATCH_SIZE" default:"100"`
	// JPGWorkerQuality defines jpg encoder quality values.
	JPGWorkerQuality int `mapstructure:"ASSETS_JPG_WORKER_JPEG_QUALITY" default:"90"`
	// JPGWorkerTimeoutMS defines worker execution timeout values in milliseconds.
	JPGWorkerTimeoutMS int `mapstructure:"ASSETS_JPG_WORKER_TIMEOUT_MS" default:"300000"`
}
