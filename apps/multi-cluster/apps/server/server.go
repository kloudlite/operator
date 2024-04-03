package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kloudlite/operator/apps/multi-cluster/apps/common"
	"github.com/kloudlite/operator/apps/multi-cluster/apps/server/env"
	"github.com/kloudlite/operator/apps/multi-cluster/constants"
	"github.com/kloudlite/operator/apps/multi-cluster/mpkg/wg"
	"github.com/kloudlite/operator/pkg/logging"
)

type server struct {
	logger logging.Logger
	app    *fiber.App
	env    *env.Env
}

func (s *server) initFunc() error {
	if err := config.load(s.env.ConfigPath); err != nil {
		return err
	}

	b, err := config.toConfigBytes()
	if err != nil {
		return err
	}

	if err := wg.ResyncWg(s.logger, b); err != nil {
		return err
	}

	return nil
}

var (
	mu sync.Mutex
)

func (s *server) Start() error {

	if err := s.initFunc(); err != nil {
		return err
	}

	go func() {
		for {
			config.cleanPeers()

			b, err := config.toConfigBytes()
			if err != nil {
				s.logger.Error(err)
				continue
			}

			wg.ResyncWg(s.logger, b)
			time.Sleep(constants.ReconDuration * time.Second)
		}
	}()

	notFound := func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNotFound)
	}

	s.app.Post("/peer", func(c *fiber.Ctx) error {
		var p common.PeerReq
		if err := p.ParseJson(c.Body()); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		pr, err := config.upsertPeer(s.logger, common.Peer{
			PublicKey: p.PublicKey,
		})

		if err != nil {
			s.logger.Error(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		presp := common.PeerResp{
			IpAddress:  fmt.Sprintf("%s/32", pr.IpAddress),
			PublicKey:  config.PublicKey,
			Endpoint:   s.env.Endpoint,
			AllowedIPs: config.getAllAllowedIPs(),
		}

		b, err := presp.ToJson()
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Send(b)
	})

	s.app.Get("/healthy", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	s.app.Get("/*", notFound)
	s.app.Post("/*", notFound)

	return nil
}
