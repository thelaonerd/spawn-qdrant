package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thelaonerd/spawn-qdrant/internal/container"
)

var (
	cfgFile string
	quiet   bool
	verbose bool
	output  string
)

var rootCmd = &cobra.Command{
	Use:   "spawn-qdrant",
	Short: "A tool to spawn Qdrant instances using Docker/Podman",
	Long: `spawn-qdrant is a CLI library for spawning and managing 
multiple Qdrant instances in Docker containers.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := container.InitRuntime(); err != nil {
			return fmt.Errorf("system error: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.spawn-qdrant.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress all diagnostic output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "text", "output format (text|json)")

	rootCmd.PersistentFlags().Int("rest-port", 6333, "Base REST port")
	rootCmd.PersistentFlags().Int("grpc-port", 6334, "Base gRPC port")

	viper.BindPFlag("rest-port", rootCmd.PersistentFlags().Lookup("rest-port"))
	viper.BindPFlag("grpc-port", rootCmd.PersistentFlags().Lookup("grpc-port"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".spawn-qdrant" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".spawn-qdrant")
	}

	viper.SetEnvPrefix("SPAWN_QDRANT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()
}

// logVerbose prints messages only if verbose flag is set and quiet is not
func logVerbose(cmd *cobra.Command, format string, a ...interface{}) {
	if !viper.GetBool("quiet") && viper.GetBool("verbose") {
		fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", a...)
	}
}

// logInfo prints diagnostic messages unless quiet is true
func logInfo(cmd *cobra.Command, format string, a ...interface{}) {
	if !viper.GetBool("quiet") {
		fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", a...)
	}
}
