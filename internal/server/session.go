package server

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/actor"
	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
)

var errNotLoggedIn = errors.New("account is not logged in")

func unsupported(call protocol.Call) (rpc.Handler[*Player], error) {
	return func(p *Player) error {
		p.Session.Logger.Warn("unimplemented client RPC", "method", call.Name)
		if p.Account == nil {
			return nil
		}
		return Acknowledge(p)
	}, nil
}

func Acknowledge(p *Player) error { return SyncTime(p) }

func SyncTime(p *Player) error {
	return p.Send("onSyncTime", calendar.ClientTime(time.Now().Unix()), 0)
}

func (s *Service) Login(p *Player, name, password string) (failure error) {
	s.mu.Lock()
	alreadyLoggedIn := p.Account != nil
	s.mu.Unlock()
	if alreadyLoggedIn {
		return errInactiveEntity
	}
	a, err := s.Store.Authenticate(name, password)
	if errors.Is(err, account.ErrAuthentication) {
		return p.Send("loginReply", 1, model.Empty{})
	}
	if errors.Is(err, account.ErrAuthenticationBusy) {
		return p.Send("loginReply", 2, model.Empty{})
	}
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	uid := a.UID
	defer func() {
		if failure != nil && s.byUID[uid] == nil {
			if err := s.Store.Social.Presence(uid, false); err != nil {
				p.Session.Logger.Error("save failed login presence", "error", err)
			}
		}
	}()
	if previous := s.byUID[a.UID]; previous != nil {
		s.retireSession(previous, false)
		_ = previous.Send("set_disconnect_error_msg", "err_be_kicked")
		previous.Session.CloseAfterFlush(time.Second)
		previous.Session.Logger.Info("session replaced", "uid", a.UID)
	}
	previous := a.LastLogin
	a, err = s.Store.Login(name)
	if err != nil {
		return err
	}
	a.PreviousLogin = previous
	if pending := a.Network.Pending; pending != nil && pending.Guild == 0 {
		if err = s.Store.Update(a, func(*sql.Tx) error { s.Store.AbandonAsyncBattle(a); return nil }); err != nil {
			return err
		}
	}
	info := model.LoginInfo{HostNumber: 1, TestServer: 1, Server: serverName, ServerTime: [2]int64{time.Now().Unix(), 0}}
	if err = p.Send("loginReply", 0, info); err != nil {
		return err
	}
	id := make([]byte, 12)
	rand.Read(id)
	profile := s.clientSnapshot(a)
	profile.LastLogin = a.PreviousLogin
	if profile.Invites, err = s.Store.JourneyInviteSnapshot(a); err != nil {
		return err
	}
	if profile.History, err = s.Store.HistorySnapshot(a); err != nil {
		return err
	}
	if profile.Guild, err = s.Store.Social.GuildSnapshot(a.UID); err != nil {
		return err
	}
	if profile.Friends, err = s.Store.Social.Snapshot(a.UID); err != nil {
		return err
	}
	properties, err := actor.NewState(profile)
	if err != nil {
		return err
	}
	if err = s.Store.Social.Presence(a.UID, true); err != nil {
		return err
	}
	payload, err := properties.Create(id, "Avatar")
	if err != nil {
		return err
	}
	session := p.Session
	session.Entity = id
	*p = Player{Account: a, properties: properties, Session: session, Level: a.Level, Day: account.LogicDay(time.Now().Unix())}
	s.byUID[a.UID] = p
	if err = session.Send(payload); err != nil {
		return err
	}
	mails, err := s.Store.Mailbox(a.UID)
	if err != nil {
		return err
	}
	if err = p.Send("upMails", mails, 1); err != nil {
		return err
	}
	if err = SyncTime(p); err != nil {
		return err
	}
	if err = p.Send("syncServerSet", 1); err != nil {
		return err
	}
	if err = p.Send("setAvatarState", 1); err != nil {
		return err
	}
	session.Logger.Info("account ready", "uid", a.UID, "heroes", len(a.Heroes), "items", len(a.Items))
	s.announcePresence(p, true)
	pending, err := s.Store.Social.Pending(a.UID)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		if err = p.Send("notifyRecentlyChat", pending); err != nil {
			return err
		}
	}
	if err = p.Send("become_player"); err != nil {
		return err
	}
	if profile.Guild.ID != 0 {
		return p.Send("onGuildInitFinish")
	}
	return nil
}
