package server

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/actor"
	"mmm-revived/internal/gate"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/social"
)

var errInactiveEntity = errors.New("RPC references an inactive entity")

type connection struct {
	service *Service
	player  *Player
	decoder protocol.Decoder
}

func (s *Service) Open(session *gate.Session) (gate.Connection, error) {
	session.Entity, session.Channel = make([]byte, 12), make([]byte, 12)
	rand.Read(session.Entity)
	rand.Read(session.Channel)
	payload, err := actor.Create(session.Entity, "LoginEntity", model.Empty{})
	if err != nil {
		return nil, err
	}
	if err = session.Send(payload); err != nil {
		return nil, err
	}
	p := &Player{Session: session}
	if err = p.Send("become_player"); err != nil {
		return nil, err
	}
	return &connection{service: s, player: p}, nil
}

func (c *connection) Receive(message gate.EntityMessage) error {
	session := c.player.Session
	if (len(message.Entity) > 0 && !bytes.Equal(message.Entity, session.Entity)) || !bytes.Equal(message.Channel, session.Channel) {
		return errInactiveEntity
	}
	calls, err := c.decoder.Decode(message.Payload)
	if err != nil {
		return err
	}
	for _, call := range calls {
		handler, public, err := c.service.routes.Resolve(call)
		if err != nil {
			malformed := fmt.Errorf("%w: %w", account.ErrProgression, err)
			handler = func(*Player) error { return malformed }
		}
		if err = c.service.invoke(c.player, call.Name, handler, public); err != nil {
			return err
		}
	}
	return nil
}

func (c *connection) Close() { c.service.disconnect(c.player) }

func (s *Service) invoke(p *Player, method string, handler rpc.Handler[*Player], public bool) error {
	if method == "login" && public {
		p.Session.Logger.Info("client RPC", "method", method)
		err := handler(p)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.flushLive()
		return s.invocationResult(p, method, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.Account != nil && s.byUID[p.Account.UID] != p {
		return gate.ErrSessionClosed
	}
	if p.Account != nil {
		p.Account.RecordGMRPC(method)
	}
	p.Session.Logger.Info("client RPC", "method", method)
	err := errNotLoggedIn
	if public || p.Account != nil {
		err = handler(p)
	}
	s.flushLive()
	return s.invocationResult(p, method, err)
}

func (s *Service) invocationResult(p *Player, method string, err error) error {
	if err == nil {
		return nil
	}
	if p.Account != nil {
		p.Account.RecordGMError(method, err.Error(), time.Now().Unix())
	}
	if errors.Is(err, account.ErrProgression) || errors.Is(err, social.ErrRejected) {
		p.Session.Logger.Warn("request rejected", "method", method, "error", err)
		if err = p.Send("showErr", 5, []int{}); err == nil {
			return nil
		}
	}
	return fmt.Errorf("RPC %s: %w", method, err)
}

func (s *Service) disconnect(p *Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retireSession(p, true)
}

func (s *Service) retireSession(p *Player, offline bool) {
	if p.Account == nil || s.byUID[p.Account.UID] != p {
		return
	}
	if pending := p.Account.Network.Pending; pending != nil && pending.Guild == 0 && pending.Token == p.Nonce {
		if err := s.Store.Update(p.Account, func(*sql.Tx) error { s.Store.AbandonAsyncBattle(p.Account); return nil }); err != nil {
			p.Session.Logger.Error("cancel disconnected asynchronous battle", "error", err)
		}
	}
	if offline {
		if err := s.Store.Social.Presence(p.Account.UID, false); err != nil {
			p.Session.Logger.Error("save offline presence", "error", err)
		}
		s.announcePresence(p, false)
	}
	delete(s.byUID, p.Account.UID)
	s.queue.Cancel(p.Account.UID)
	if p.Room != nil {
		if err := s.FinishMatch(p.Room, p.Opponent().Account.UID); err != nil {
			p.Session.Logger.Error("finish match on disconnect", "error", err)
		}
	}
}
