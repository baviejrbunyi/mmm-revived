package server

func (s *Service) expireCosmetics(now int64) {
	for _, p := range s.byUID {
		if p.properties == nil || !p.Account.ExpireCosmetics(s.Store.Catalog, now) {
			continue
		}
		if err := s.SaveAndSync(p, "makeupComp"); err != nil {
			p.Session.Logger.Error("expire timed cosmetics", "uid", p.Account.UID, "error", err)
		}
	}
}
