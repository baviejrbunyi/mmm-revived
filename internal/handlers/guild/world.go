package guild

import (
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/calendar"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
	"mmm-revived/internal/social"
)

const worldChannel = "local-1"

type worldSender struct {
	UID   string `json:"uid"`
	Head  string `json:"head"`
	Frame string `json:"headFrame"`
	Name  string `json:"name"`
}

type worldMessage struct {
	ID        string      `json:"messageId"`
	Poster    string      `json:"poster"`
	Text      string      `json:"message"`
	Kind      int         `json:"msgType"`
	Timestamp int64       `json:"ts"`
	Channel   string      `json:"channelId"`
	Extra     worldSender `json:"extra"`
}

type worldList struct {
	Messages []string `json:"messages"`
	Append   bool     `json:"append"`
}

func worldReply[T any](value T) (string, error) {
	data, err := json.Marshal(struct {
		Code int `json:"error_code"`
		Data T   `json:"data"`
	}{0, value})
	return string(data), err
}

func (s service) worldSenders(messages []social.WorldMessage) (map[model.UserID]account.Account, error) {
	result := map[model.UserID]account.Account{}
	var missing []model.UserID
	for _, message := range messages {
		if _, known := result[message.Sender]; known || slices.Contains(missing, message.Sender) {
			continue
		}
		if live := s.Online(message.Sender); live != nil {
			result[message.Sender] = *live.Account
		} else {
			missing = append(missing, message.Sender)
		}
	}
	for batch := range slices.Chunk(missing, 100) {
		profiles, err := s.Store.NetworkProfiles(batch)
		if err != nil {
			return nil, err
		}
		for uid, profile := range profiles {
			result[uid] = profile.Account
		}
	}
	return result, nil
}

func (s service) encodeWorld(messages []social.WorldMessage) ([]string, error) {
	senders, err := s.worldSenders(messages)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(messages))
	for _, message := range messages {
		sender := senders[message.Sender]
		uid := strconv.FormatUint(uint64(message.Sender), 10)
		name := sender.TeamName
		if name == "" {
			name = sender.Name
		}
		data, err := json.Marshal(worldMessage{ID: strconv.FormatInt(message.ID, 10), Poster: uid, Text: message.Text, Kind: message.Kind, Timestamp: calendar.ClientTime(message.Created) * 1000, Channel: worldChannel, Extra: worldSender{UID: uid, Head: strconv.FormatInt(int64(sender.Profile.Head), 10), Frame: strconv.FormatInt(int64(sender.Profile.Frame), 10), Name: name}})
		if err != nil {
			return nil, err
		}
		result = append(result, string(data))
	}
	return result, nil
}

func (s service) listWorld(p *server.Player) error {
	history, err := s.Store.Social.WorldHistory()
	if err != nil {
		return err
	}
	messages, err := s.encodeWorld(history)
	if err != nil {
		return err
	}
	reply, err := worldReply(worldList{Messages: messages})
	if err != nil {
		return err
	}
	return p.Send("onListWorldChannel", reply)
}

func SendWorldChat(srv *server.Service, p *server.Player, kind int, text string) error {
	return service{srv}.sendWorldChat(p, kind, text)
}

func (s service) sendWorldChat(p *server.Player, kind int, text string) error {
	message, err := s.Store.Social.SendWorldChat(p.Account.UID, kind, text, time.Now().Unix())
	if err != nil {
		return Reject(p, err)
	}
	encoded, err := s.encodeWorld([]social.WorldMessage{message})
	if err != nil {
		return err
	}
	ack, err := worldReply(map[string]string{"messageId": strconv.FormatInt(message.ID, 10)})
	if err != nil {
		return err
	}
	if err = p.Send("onSendWorldChat", ack); err != nil {
		return err
	}
	broadcast, err := worldReply(worldList{Messages: encoded, Append: true})
	if err != nil {
		return err
	}
	members, err := s.Store.Social.WorldMembers()
	if err != nil {
		return err
	}
	for _, uid := range members {
		s.notify(uid, "onListWorldChannel", broadcast)
	}
	return nil
}
