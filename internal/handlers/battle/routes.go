package battle

import (
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type Prepare func(*server.Service, *server.Player, CreateBattle) (bool, error)

type Lock func(*server.Service, *server.Player, []model.HeroID) error

type Finish func(*server.Service, *server.Player, VerifyBattle) (bool, error)

type Modes struct {
	Prepare []Prepare
	Lock    []Lock
	Finish  []Finish
}

type service struct {
	*server.Service
	modes Modes
}

func Register(srv *server.Service, r *server.Router, modes Modes) {
	s := service{srv, modes}
	r.Handle("createBattle", rpc.Bind(DecodeCreateBattle, s.createBattle))
	r.Handle("lockHero", rpc.Bind(DecodeDeployment, s.lockHeroes))
	r.Handle("battleVerify", rpc.Bind(DecodeVerifyBattle, s.verifyBattle))
}
