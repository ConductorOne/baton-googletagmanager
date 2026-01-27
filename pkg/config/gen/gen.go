package main

import (
	cfg "github.com/conductorone/baton-googletagmanager/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("googletagmanager", cfg.Config)
}
