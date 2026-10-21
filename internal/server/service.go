package server

import (
	"cmp"
	"sync"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/matchmaking"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
)

const serverName = "mmm-revived"

const (
	DefaultTickInterval  = time.Second
	DefaultQueueTimeout  = 2 * time.Minute
	DefaultLoadTimeout   = 2 * time.Minute
	DefaultCombatTimeout = 15 * time.Minute
)

type Config struct {
	TickInterval  time.Duration
	QueueTimeout  time.Duration
	QueueCapacity int
	LoadTimeout   time.Duration
	CombatTimeout time.Duration
}

func (c Config) normalized() Config {
	c.TickInterval = cmp.Or(max(c.TickInterval, 0), DefaultTickInterval)
	c.QueueTimeout = cmp.Or(max(c.QueueTimeout, 0), DefaultQueueTimeout)
	c.QueueCapacity = cmp.Or(max(c.QueueCapacity, 0), matchmaking.DefaultCapacity)
	c.LoadTimeout = cmp.Or(max(c.LoadTimeout, 0), DefaultLoadTimeout)
	c.CombatTimeout = cmp.Or(max(c.CombatTimeout, 0), DefaultCombatTimeout)
	return c
}

type Router = rpc.Router[*Player]

type Service struct {
	Store       *account.Store
	config      Config
	routes      *Router
	mu          sync.Mutex
	byUID       map[model.UserID]*Player
	queue       matchmaking.Queue
	rooms       map[string]*Room
	catalogSets func() catalogSets
}

func NewService(store *account.Store, config Config, register func(*Service, *Router)) *Service {
	config = config.normalized()
	s := &Service{Store: store, config: config, byUID: map[model.UserID]*Player{}, rooms: map[string]*Room{}, queue: matchmaking.Queue{Capacity: config.QueueCapacity}}
	s.catalogSets = sync.OnceValue(func() catalogSets { return newCatalogSets(s.Store.Catalog) })
	s.routes = rpc.NewRouter(unsupported)
	register(s, s.routes)
	return s
}

func (s *Service) Online(uid model.UserID) *Player { return s.byUID[uid] }

func (s *Service) Queue() *matchmaking.Queue { return &s.queue }

func (s *Service) Config() Config { return s.config }

func (s *Service) OpenRoom(match *Room) { s.rooms[match.ID] = match }
