package manager

import (
	"context"
	"sync"
	"time"

	"github.com/Filecoin-Titan/titan-container/api"
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/pkg/errors"
)

var HeartbeatInterval = 10 * time.Second

var ProviderTTL = 30 * time.Second

var (
	ErrProviderNotExist = errors.New("provider not exist")
)

type ProviderManager struct {
	lk        sync.RWMutex
	providers map[types.ProviderID]*providerLife
}

type providerLife struct {
	api.Provider
	LastSeen   time.Time
	remoteAddr string
}

func (p *providerLife) Update() {
	p.LastSeen = time.Now()
}

func (p *providerLife) Expired() bool {
	return p.LastSeen.Add(ProviderTTL).Before(time.Now())
}

func NewProviderScheduler() *ProviderManager {
	s := &ProviderManager{
		providers: make(map[types.ProviderID]*providerLife),
	}

	go s.watch()
	return s
}

func (p *ProviderManager) AddProvider(id types.ProviderID, providerApi api.Provider, remoteAddr string) error {
	p.lk.Lock()
	defer p.lk.Unlock()

	_, exist := p.providers[id]
	if exist {
		return nil
	}

	p.providers[id] = &providerLife{
		Provider:   providerApi,
		LastSeen:   time.Now(),
		remoteAddr: remoteAddr,
	}
	return nil
}

func (p *ProviderManager) CloseProvider(id types.ProviderID) error {
	p.lk.Lock()
	defer p.lk.Unlock()

	oldProvider, exist := p.providers[id]
	if !exist {
		return nil
	}

	if provider, ok := oldProvider.Provider.(remoteProvider); ok {
		provider.Close()
	} else {
		return errors.New("can not convert provider to remoteProvider")
	}

	delete(p.providers, id)

	return nil
}

func (p *ProviderManager) Get(id types.ProviderID) (api.Provider, error) {
	p.lk.Lock()
	defer p.lk.Unlock()

	provider, exist := p.providers[id]
	if !exist {
		log.Infof("p.providers:%#v", p.providers)
		return nil, ErrProviderNotExist
	}

	return provider, nil
}

func (p *ProviderManager) GetRemoteAddr(id types.ProviderID) (string, error) {
	p.lk.Lock()
	defer p.lk.Unlock()

	provider, exist := p.providers[id]
	if !exist {
		log.Infof("p.providers:%#v", p.providers)
		return "", ErrProviderNotExist
	}

	return provider.remoteAddr, nil
}

func (p *ProviderManager) delProvider(id types.ProviderID) {
	p.lk.Lock()
	defer p.lk.Unlock()
	delete(p.providers, id)
}

func (p *ProviderManager) watch() {
	heartbeatTimer := time.NewTicker(HeartbeatInterval)
	defer heartbeatTimer.Stop()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for {
		select {
		case <-heartbeatTimer.C:
		}

		p.lk.Lock()
		for id, provider := range p.providers {
			sctx, scancel := context.WithTimeout(ctx, HeartbeatInterval/2)
			_, err := provider.Session(sctx)
			scancel()
			if err != nil {
				if !provider.Expired() {
					// Likely temporary error
					log.Warnw("failed to check provider session", "error", err)
					continue
				}

				log.Warnw("Provider closing", "ProviderID", id)
				delete(p.providers, id)
				continue
			}
			provider.Update()
		}
		p.lk.Unlock()

	}
}
