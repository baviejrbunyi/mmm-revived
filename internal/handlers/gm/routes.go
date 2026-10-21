package gm

import (
	"database/sql"
	"errors"

	"mmm-revived/internal/handlers/system"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("gmcmd", rpc.Bind(DecodeGMCommand, func(p *server.Player, command GMCommand) error {
		if s.Store.Rules == nil {
			return s.gmFailure(p, "gmcmd", errGMRules)
		}
		if command.Ship != nil {
			if command.Recharge != "" {
				product, known := s.Store.Rules.System.Recharges[command.Recharge]
				if !known {
					return s.gmFailure(p, "gmcmd", errors.New("unknown recharge "+command.Recharge))
				}
				command.Ship.Product = product
			}
			return system.Ship(s.Service, p, *command.Ship)
		}
		if command.Validation != "" {
			return s.gmFailure(p, "gmcmd", errors.New(command.Validation))
		}
		return s.executeGM(p, command.Command)
	}))
	r.Handle("gmgeterror", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		return p.Send("onGmGetError", p.Account.GM.LastError)
	}))
	r.Handle("getGmSdkToken", rpc.Bind(DecodeGMTokenRequest, func(p *server.Player, _ struct{}) error {
		return p.Send("onGetGmSdkToken", "")
	}))
	r.Handle("gmGenFilePickerToken", rpc.Bind(DecodeGMUploadURL, func(p *server.Player, _ string) error {
		return p.Send("onGmGenFilePickerToken", "")
	}))
	r.Handle("gmServerCoverageStart", rpc.Bind(DecodeCoverageUser, func(p *server.Player, user string) error {
		return s.gmDiagnostic(p, func(state *model.GMState, now int64) (string, error) {
			if state.Coverage.Enabled && state.Coverage.User == user {
				return "RPC coverage running", nil
			}
			state.Coverage.Enabled = true
			state.Coverage.User, state.Coverage.Started, state.Coverage.Methods = user, now, map[string]int64{}
			return "RPC coverage started", nil
		})
	}))
	r.Handle("recordGmStr", rpc.Bind(DecodeGMHistory, func(p *server.Player, history string) error {
		if p.Account.GM.History == history {
			return server.Acknowledge(p)
		}
		if err := s.Store.Update(p.Account, func(*sql.Tx) error { return p.Account.RecordGMHistory(history) }); err != nil {
			return err
		}
		return p.Update([]string{"gmComp"}, model.GMComponent{History: p.Account.GM.History})
	}))
}
