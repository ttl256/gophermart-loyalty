package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/alexflint/go-arg"
)

type Server struct {
	Address        string     `arg:"-a,env:RUN_ADDRESS"`
	DSN            string     `arg:"-d,env:DATABASE_URI"`
	AccrualAddress string     `arg:"-r,env:ACCRUAL_SYSTEM_ADDRESS"`
	Secret         string     `arg:"-s,env:SECRET"`
	LogLevel       slog.Level `arg:"--loglevel,env:LOG_LEVEL"`
}

func NewServer() *Server {
	return &Server{
		Address:        "localhost:8080",
		DSN:            "",
		AccrualAddress: "",
		Secret:         "",
		LogLevel:       slog.LevelInfo,
	}
}

func BuildConfig(args []string, helpOut io.Writer) (*Server, error) {
	cfg := NewServer()
	parser, err := arg.NewParser(
		arg.Config{
			Program:           "",
			IgnoreEnv:         false,
			IgnoreDefault:     false,
			StrictSubcommands: false,
			EnvPrefix:         "",
			Exit:              os.Exit,
			Out:               helpOut,
		},
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("definiton of args struct: %w", err)
	}
	err = parser.Parse(args)
	if err != nil {
		if errors.Is(err, arg.ErrHelp) {
			parser.WriteHelp(helpOut)
			return nil, arg.ErrHelp
		}
		return nil, fmt.Errorf("parsing arguments: %w", err)
	}

	return cfg, nil
}
