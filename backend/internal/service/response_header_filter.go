package service

import (
	"github.com/Wei-Shaw/nbapi/internal/config"
	"github.com/Wei-Shaw/nbapi/internal/util/responseheaders"
)

func compileResponseHeaderFilter(cfg *config.Config) *responseheaders.CompiledHeaderFilter {
	if cfg == nil {
		return nil
	}
	return responseheaders.CompileHeaderFilter(cfg.Security.ResponseHeaders)
}
