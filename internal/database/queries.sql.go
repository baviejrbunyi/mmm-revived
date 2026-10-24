package database

import (
	"context"
	"database/sql"

	"mmm-revived/internal/model"
)

const AccountExists = `
SELECT count(*) FROM accounts WHERE uid=?1
`

func (q *Queries) AccountExists(ctx context.Context, uid model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.accountExistsStmt, AccountExists, uid)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const AcknowledgeAllMessages = `
UPDATE chat_messages SET acknowledged=1 WHERE receiver=?1 AND channel=1 AND acknowledged=0
`

func (q *Queries) AcknowledgeAllMessages(ctx context.Context, receiver model.UserID) error {
	_, err := q.exec(ctx, q.acknowledgeAllMessagesStmt, AcknowledgeAllMessages, receiver)
	return err
}

const AcknowledgePairMessages = `
UPDATE chat_messages SET acknowledged=1 WHERE channel=1 AND ((sender=?1 AND receiver=?2) OR (sender=?2 AND receiver=?1))
`

type AcknowledgePairMessagesParams struct {
	Sender   model.UserID
	Receiver model.UserID
}

func (q *Queries) AcknowledgePairMessages(ctx context.Context, arg AcknowledgePairMessagesParams) error {
	_, err := q.exec(ctx, q.acknowledgePairMessagesStmt, AcknowledgePairMessages, arg.Sender, arg.Receiver)
	return err
}

const AcknowledgePeerMessages = `
UPDATE chat_messages SET acknowledged=1 WHERE receiver=?1 AND sender=?2 AND channel=1 AND acknowledged=0
`

type AcknowledgePeerMessagesParams struct {
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) AcknowledgePeerMessages(ctx context.Context, arg AcknowledgePeerMessagesParams) error {
	_, err := q.exec(ctx, q.acknowledgePeerMessagesStmt, AcknowledgePeerMessages, arg.Receiver, arg.Sender)
	return err
}

const AddGuildContribution = `
INSERT INTO guild_contributions(guild_id,uid,week,score) VALUES (?,?,?,?) ON CONFLICT(guild_id,week,uid) DO UPDATE SET score=score+excluded.score
`

type AddGuildContributionParams struct {
	GuildID model.GuildID
	UID     model.UserID
	Week    int64
	Score   int64
}

func (q *Queries) AddGuildContribution(ctx context.Context, arg AddGuildContributionParams) error {
	_, err := q.exec(ctx, q.addGuildContributionStmt, AddGuildContribution,
		arg.GuildID,
		arg.UID,
		arg.Week,
		arg.Score,
	)
	return err
}

const AddGuildHistoryScore = `
UPDATE guild_members SET history_score=history_score+?1 WHERE uid=?2
`

type AddGuildHistoryScoreParams struct {
	Score int64
	UID   model.UserID
}

func (q *Queries) AddGuildHistoryScore(ctx context.Context, arg AddGuildHistoryScoreParams) error {
	_, err := q.exec(ctx, q.addGuildHistoryScoreStmt, AddGuildHistoryScore, arg.Score, arg.UID)
	return err
}

const AddGuildWarProgress = `
INSERT INTO guild_war_progress(guild_id,war,uid,area,score,assist,score_time,assist_time) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(guild_id,war,uid,area) DO UPDATE SET score=score+excluded.score, assist=assist+excluded.assist, score_time=max(score_time,excluded.score_time), assist_time=max(assist_time,excluded.assist_time)
`

type AddGuildWarProgressParams struct {
	GuildID    model.GuildID
	War        int64
	UID        model.UserID
	Area       int64
	Score      int64
	Assist     int64
	ScoreTime  int64
	AssistTime int64
}

func (q *Queries) AddGuildWarProgress(ctx context.Context, arg AddGuildWarProgressParams) error {
	_, err := q.exec(ctx, q.addGuildWarProgressStmt, AddGuildWarProgress,
		arg.GuildID,
		arg.War,
		arg.UID,
		arg.Area,
		arg.Score,
		arg.Assist,
		arg.ScoreTime,
		arg.AssistTime,
	)
	return err
}

const AddSupportUsage = `
INSERT INTO support_usage(owner,borrower,hero,day) VALUES (?,?,?,?) ON CONFLICT(owner,borrower,hero,day) DO UPDATE SET uses=uses+1
`

type AddSupportUsageParams struct {
	Owner    model.UserID
	Borrower model.UserID
	Hero     model.HeroID
	Day      int64
}

func (q *Queries) AddSupportUsage(ctx context.Context, arg AddSupportUsageParams) error {
	_, err := q.exec(ctx, q.addSupportUsageStmt, AddSupportUsage,
		arg.Owner,
		arg.Borrower,
		arg.Hero,
		arg.Day,
	)
	return err
}

const BlockEither = `
SELECT count(*) FROM blocks WHERE (owner=?1 AND peer=?2) OR (owner=?2 AND peer=?1)
`

type BlockEitherParams struct {
	Owner model.UserID
	Peer  model.UserID
}

func (q *Queries) BlockEither(ctx context.Context, arg BlockEitherParams) (int64, error) {
	row := q.queryRow(ctx, q.blockEitherStmt, BlockEither, arg.Owner, arg.Peer)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const ClaimMail = `
UPDATE mailbox SET status=3 WHERE owner=?1 AND id=?2 AND status<3 AND (expires=0 OR expires>?3)
`

type ClaimMailParams struct {
	Owner model.UserID
	ID    model.MailID
	Now   int64
}

func (q *Queries) ClaimMail(ctx context.Context, arg ClaimMailParams) (int64, error) {
	result, err := q.exec(ctx, q.claimMailStmt, ClaimMail, arg.Owner, arg.ID, arg.Now)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const ClaimSupportUsage = `
UPDATE support_usage SET claimed=uses WHERE owner=? AND claimed<uses
`

func (q *Queries) ClaimSupportUsage(ctx context.Context, owner model.UserID) (int64, error) {
	result, err := q.exec(ctx, q.claimSupportUsageStmt, ClaimSupportUsage, owner)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const ClaimableGift = `
SELECT count(*) FROM friend_gifts g WHERE g.day=?1 AND g.receiver=?2 AND g.sender=?3 AND g.claimed=0 AND EXISTS(SELECT 1 FROM friends f WHERE f.owner=g.receiver AND f.peer=g.sender)
`

type ClaimableGiftParams struct {
	Day      int64
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) ClaimableGift(ctx context.Context, arg ClaimableGiftParams) (int64, error) {
	row := q.queryRow(ctx, q.claimableGiftStmt, ClaimableGift, arg.Day, arg.Receiver, arg.Sender)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const CountWorldMembers = `
SELECT count(*) FROM world_membership WHERE channel=?
`

func (q *Queries) CountWorldMembers(ctx context.Context, channel string) (int64, error) {
	row := q.queryRow(ctx, q.countWorldMembersStmt, CountWorldMembers, channel)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const CreateGuild = `
INSERT INTO guilds(name,search_name,signature,announcement,approval,minimum_level,head,frame,boss,created,name_updated) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?10) RETURNING id
`

type CreateGuildParams struct {
	Name         string
	SearchName   string
	Signature    string
	Announcement string
	Approval     int64
	MinimumLevel int64
	Head         int64
	Frame        int64
	Boss         model.UserID
	Created      int64
}

func (q *Queries) CreateGuild(ctx context.Context, arg CreateGuildParams) (model.GuildID, error) {
	row := q.queryRow(ctx, q.createGuildStmt, CreateGuild,
		arg.Name,
		arg.SearchName,
		arg.Signature,
		arg.Announcement,
		arg.Approval,
		arg.MinimumLevel,
		arg.Head,
		arg.Frame,
		arg.Boss,
		arg.Created,
	)
	var id model.GuildID
	err := row.Scan(&id)
	return id, err
}

const DeleteAllClaimedMail = `
DELETE FROM mailbox WHERE owner=?1 AND (status=3 OR (expires>0 AND expires<=?2))
`

type DeleteAllClaimedMailParams struct {
	Owner model.UserID
	Now   int64
}

func (q *Queries) DeleteAllClaimedMail(ctx context.Context, arg DeleteAllClaimedMailParams) error {
	_, err := q.exec(ctx, q.deleteAllClaimedMailStmt, DeleteAllClaimedMail, arg.Owner, arg.Now)
	return err
}

const DeleteAllFriendRequests = `
DELETE FROM friend_requests WHERE receiver=?1
`

func (q *Queries) DeleteAllFriendRequests(ctx context.Context, receiver model.UserID) error {
	_, err := q.exec(ctx, q.deleteAllFriendRequestsStmt, DeleteAllFriendRequests, receiver)
	return err
}

const DeleteBlock = `
DELETE FROM blocks WHERE owner=?1 AND peer=?2
`

type DeleteBlockParams struct {
	Owner model.UserID
	Peer  model.UserID
}

func (q *Queries) DeleteBlock(ctx context.Context, arg DeleteBlockParams) error {
	_, err := q.exec(ctx, q.deleteBlockStmt, DeleteBlock, arg.Owner, arg.Peer)
	return err
}

const DeleteClaimedMail = `
DELETE FROM mailbox WHERE owner=?1 AND id=?2 AND (status=3 OR has_reward=0 OR (expires>0 AND expires<=?3))
`

type DeleteClaimedMailParams struct {
	Owner model.UserID
	ID    model.MailID
	Now   int64
}

func (q *Queries) DeleteClaimedMail(ctx context.Context, arg DeleteClaimedMailParams) (int64, error) {
	result, err := q.exec(ctx, q.deleteClaimedMailStmt, DeleteClaimedMail, arg.Owner, arg.ID, arg.Now)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const DeleteFriendPair = `
DELETE FROM friends WHERE (owner=?1 AND peer=?2) OR (owner=?2 AND peer=?1)
`

type DeleteFriendPairParams struct {
	Owner model.UserID
	Peer  model.UserID
}

func (q *Queries) DeleteFriendPair(ctx context.Context, arg DeleteFriendPairParams) error {
	_, err := q.exec(ctx, q.deleteFriendPairStmt, DeleteFriendPair, arg.Owner, arg.Peer)
	return err
}

const DeleteFriendRequest = `
DELETE FROM friend_requests WHERE receiver=?1 AND sender=?2
`

type DeleteFriendRequestParams struct {
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) DeleteFriendRequest(ctx context.Context, arg DeleteFriendRequestParams) error {
	_, err := q.exec(ctx, q.deleteFriendRequestStmt, DeleteFriendRequest, arg.Receiver, arg.Sender)
	return err
}

const DeleteFriendRequestPair = `
DELETE FROM friend_requests WHERE (receiver=?1 AND sender=?2) OR (receiver=?2 AND sender=?1)
`

type DeleteFriendRequestPairParams struct {
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) DeleteFriendRequestPair(ctx context.Context, arg DeleteFriendRequestPairParams) error {
	_, err := q.exec(ctx, q.deleteFriendRequestPairStmt, DeleteFriendRequestPair, arg.Receiver, arg.Sender)
	return err
}

const DeleteGuild = `
DELETE FROM guilds WHERE id=?1
`

func (q *Queries) DeleteGuild(ctx context.Context, id model.GuildID) error {
	_, err := q.exec(ctx, q.deleteGuildStmt, DeleteGuild, id)
	return err
}

const DeleteGuildApplication = `
DELETE FROM guild_applications WHERE guild_id=?1 AND applicant=?2
`

type DeleteGuildApplicationParams struct {
	GuildID   model.GuildID
	Applicant model.UserID
}

func (q *Queries) DeleteGuildApplication(ctx context.Context, arg DeleteGuildApplicationParams) error {
	_, err := q.exec(ctx, q.deleteGuildApplicationStmt, DeleteGuildApplication, arg.GuildID, arg.Applicant)
	return err
}

const DeleteGuildContributions = `
DELETE FROM guild_contributions WHERE guild_id=? AND week<?
`

type DeleteGuildContributionsParams struct {
	GuildID model.GuildID
	Week    int64
}

func (q *Queries) DeleteGuildContributions(ctx context.Context, arg DeleteGuildContributionsParams) error {
	_, err := q.exec(ctx, q.deleteGuildContributionsStmt, DeleteGuildContributions, arg.GuildID, arg.Week)
	return err
}

const DeleteGuildInvite = `
DELETE FROM guild_invites WHERE guild_id=?1 AND player=?2
`

type DeleteGuildInviteParams struct {
	GuildID model.GuildID
	Player  model.UserID
}

func (q *Queries) DeleteGuildInvite(ctx context.Context, arg DeleteGuildInviteParams) error {
	_, err := q.exec(ctx, q.deleteGuildInviteStmt, DeleteGuildInvite, arg.GuildID, arg.Player)
	return err
}

const DeleteGuildMember = `
DELETE FROM guild_members WHERE uid=?1
`

func (q *Queries) DeleteGuildMember(ctx context.Context, uid model.UserID) error {
	_, err := q.exec(ctx, q.deleteGuildMemberStmt, DeleteGuildMember, uid)
	return err
}

const DeletePlayerGuildApplications = `
DELETE FROM guild_applications WHERE applicant=?1
`

func (q *Queries) DeletePlayerGuildApplications(ctx context.Context, applicant model.UserID) error {
	_, err := q.exec(ctx, q.deletePlayerGuildApplicationsStmt, DeletePlayerGuildApplications, applicant)
	return err
}

const DeletePlayerGuildInvites = `
DELETE FROM guild_invites WHERE player=?1
`

func (q *Queries) DeletePlayerGuildInvites(ctx context.Context, player model.UserID) error {
	_, err := q.exec(ctx, q.deletePlayerGuildInvitesStmt, DeletePlayerGuildInvites, player)
	return err
}

const DeleteRecord = `
DELETE FROM account_records WHERE uid=?1 AND category=?2 AND id=?3
`

type DeleteRecordParams struct {
	UID      model.UserID
	Category model.RecordCategory
	ID       string
}

func (q *Queries) DeleteRecord(ctx context.Context, arg DeleteRecordParams) error {
	_, err := q.exec(ctx, q.deleteRecordStmt, DeleteRecord, arg.UID, arg.Category, arg.ID)
	return err
}

const DeliverMail = `
INSERT INTO mailbox(owner,title,content,sender,template,created,expires,reward_id,items,has_reward) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10) RETURNING id
`

type DeliverMailParams struct {
	Owner     model.UserID
	Title     []byte
	Content   []byte
	Sender    []byte
	Template  int64
	Created   int64
	Expires   int64
	RewardID  model.RewardID
	Items     []byte
	HasReward int64
}

func (q *Queries) DeliverMail(ctx context.Context, arg DeliverMailParams) (model.MailID, error) {
	row := q.queryRow(ctx, q.deliverMailStmt, DeliverMail,
		arg.Owner,
		arg.Title,
		arg.Content,
		arg.Sender,
		arg.Template,
		arg.Created,
		arg.Expires,
		arg.RewardID,
		arg.Items,
		arg.HasReward,
	)
	var id model.MailID
	err := row.Scan(&id)
	return id, err
}

const FindGuildsByName = `
SELECT id FROM guilds WHERE search_name GLOB ?1 ORDER BY search_name,id LIMIT 20
`

func (q *Queries) FindGuildsByName(ctx context.Context, pattern string) ([]model.GuildID, error) {
	rows, err := q.query(ctx, q.findGuildsByNameStmt, FindGuildsByName, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.GuildID{}
	for rows.Next() {
		var id model.GuildID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const FindPlayerByID = `
SELECT a.uid FROM accounts a WHERE a.uid=?1 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=a.uid AND b.peer=?2) LIMIT 1
`

type FindPlayerByIDParams struct {
	Target model.UserID
	Self   model.UserID
}

func (q *Queries) FindPlayerByID(ctx context.Context, arg FindPlayerByIDParams) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.findPlayerByIDStmt, FindPlayerByID, arg.Target, arg.Self)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const FindPlayersByName = `
SELECT a.uid FROM accounts a WHERE NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=a.uid AND b.peer=?1) AND a.search_name GLOB ?2 ORDER BY a.search_name,a.uid LIMIT 20
`

type FindPlayersByNameParams struct {
	Self    model.UserID
	Pattern string
}

func (q *Queries) FindPlayersByName(ctx context.Context, arg FindPlayersByNameParams) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.findPlayersByNameStmt, FindPlayersByName, arg.Self, arg.Pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const FriendCount = `
SELECT count(*) FROM friends WHERE owner=?1
`

func (q *Queries) FriendCount(ctx context.Context, owner model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.friendCountStmt, FriendCount, owner)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const FriendRequestExists = `
SELECT count(*) FROM friend_requests WHERE receiver=?1 AND sender=?2
`

type FriendRequestExistsParams struct {
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) FriendRequestExists(ctx context.Context, arg FriendRequestExistsParams) (int64, error) {
	row := q.queryRow(ctx, q.friendRequestExistsStmt, FriendRequestExists, arg.Receiver, arg.Sender)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const FriendshipExists = `
SELECT count(*) FROM friends WHERE owner=?1 AND peer=?2
`

type FriendshipExistsParams struct {
	Owner model.UserID
	Peer  model.UserID
}

func (q *Queries) FriendshipExists(ctx context.Context, arg FriendshipExistsParams) (int64, error) {
	row := q.queryRow(ctx, q.friendshipExistsStmt, FriendshipExists, arg.Owner, arg.Peer)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const GetAccount = `
SELECT uid,profile FROM accounts WHERE login=?1
`

type GetAccountRow struct {
	UID     model.UserID
	Profile []byte
}

func (q *Queries) GetAccount(ctx context.Context, login string) (GetAccountRow, error) {
	row := q.queryRow(ctx, q.getAccountStmt, GetAccount, login)
	var i GetAccountRow
	err := row.Scan(&i.UID, &i.Profile)
	return i, err
}

const GetAccountLevel = `
SELECT level FROM accounts WHERE uid=?
`

func (q *Queries) GetAccountLevel(ctx context.Context, uid model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.getAccountLevelStmt, GetAccountLevel, uid)
	var level int64
	err := row.Scan(&level)
	return level, err
}

const GetGuild = `
SELECT id, name, search_name, signature, announcement, approval, minimum_level, head, frame, boss, level, experience, member_count, created, name_updated FROM guilds WHERE id=?1
`

func (q *Queries) GetGuild(ctx context.Context, id model.GuildID) (Guild, error) {
	row := q.queryRow(ctx, q.getGuildStmt, GetGuild, id)
	var i Guild
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.SearchName,
		&i.Signature,
		&i.Announcement,
		&i.Approval,
		&i.MinimumLevel,
		&i.Head,
		&i.Frame,
		&i.Boss,
		&i.Level,
		&i.Experience,
		&i.MemberCount,
		&i.Created,
		&i.NameUpdated,
	)
	return i, err
}

const GetGuildApplication = `
SELECT created FROM guild_applications WHERE guild_id=?1 AND applicant=?2
`

type GetGuildApplicationParams struct {
	GuildID   model.GuildID
	Applicant model.UserID
}

func (q *Queries) GetGuildApplication(ctx context.Context, arg GetGuildApplicationParams) (int64, error) {
	row := q.queryRow(ctx, q.getGuildApplicationStmt, GetGuildApplication, arg.GuildID, arg.Applicant)
	var created int64
	err := row.Scan(&created)
	return created, err
}

const GetGuildInvite = `
SELECT created FROM guild_invites WHERE guild_id=?1 AND player=?2
`

type GetGuildInviteParams struct {
	GuildID model.GuildID
	Player  model.UserID
}

func (q *Queries) GetGuildInvite(ctx context.Context, arg GetGuildInviteParams) (int64, error) {
	row := q.queryRow(ctx, q.getGuildInviteStmt, GetGuildInvite, arg.GuildID, arg.Player)
	var created int64
	err := row.Scan(&created)
	return created, err
}

const GetGuildMembership = `
SELECT uid, guild_id, role, joined, score, history_score FROM guild_members WHERE uid=?1
`

func (q *Queries) GetGuildMembership(ctx context.Context, uid model.UserID) (GuildMember, error) {
	row := q.queryRow(ctx, q.getGuildMembershipStmt, GetGuildMembership, uid)
	var i GuildMember
	err := row.Scan(
		&i.UID,
		&i.GuildID,
		&i.Role,
		&i.Joined,
		&i.Score,
		&i.HistoryScore,
	)
	return i, err
}

const GetGuildPlayerState = `
SELECT uid, last_left, last_search FROM guild_player_state WHERE uid=?1
`

func (q *Queries) GetGuildPlayerState(ctx context.Context, uid model.UserID) (GuildPlayerState, error) {
	row := q.queryRow(ctx, q.getGuildPlayerStateStmt, GetGuildPlayerState, uid)
	var i GuildPlayerState
	err := row.Scan(&i.UID, &i.LastLeft, &i.LastSearch)
	return i, err
}

const GetGuildWar = `
SELECT state FROM guild_wars WHERE guild_id=? AND war=?
`

type GetGuildWarParams struct {
	GuildID model.GuildID
	War     int64
}

func (q *Queries) GetGuildWar(ctx context.Context, arg GetGuildWarParams) ([]byte, error) {
	row := q.queryRow(ctx, q.getGuildWarStmt, GetGuildWar, arg.GuildID, arg.War)
	var state []byte
	err := row.Scan(&state)
	return state, err
}

const GetGuilds = `
SELECT id, name, search_name, signature, announcement, approval, minimum_level, head, frame, boss, level, experience, member_count, created, name_updated FROM guilds WHERE id IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(?1 AS TEXT)))
`

func (q *Queries) GetGuilds(ctx context.Context, ids string) ([]Guild, error) {
	rows, err := q.query(ctx, q.getGuildsStmt, GetGuilds, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Guild{}
	for rows.Next() {
		var i Guild
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.SearchName,
			&i.Signature,
			&i.Announcement,
			&i.Approval,
			&i.MinimumLevel,
			&i.Head,
			&i.Frame,
			&i.Boss,
			&i.Level,
			&i.Experience,
			&i.MemberCount,
			&i.Created,
			&i.NameUpdated,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetInviteProgress = `
SELECT l.invitee,l.bound,a.level,
CAST(EXISTS(SELECT 1 FROM account_records p WHERE p.uid=l.invitee AND p.category='passed' AND p.id='100109' AND coalesce(json_extract(p.payload,'$.stars'),0)>0) AS INTEGER) AS story,
CAST(coalesce(json_extract(t1.payload,'$.status'),0) AS INTEGER) AS purchase1,
CAST(coalesce(json_extract(t2.payload,'$.status'),0) AS INTEGER) AS purchase2,
CAST(coalesce(json_extract(t3.payload,'$.status'),0) AS INTEGER) AS purchase3
FROM invite_links l JOIN accounts a ON a.uid=l.invitee
LEFT JOIN account_records t1 ON t1.uid=l.invitee AND t1.category='task' AND t1.id='3711'
LEFT JOIN account_records t2 ON t2.uid=l.invitee AND t2.category='task' AND t2.id='3712'
LEFT JOIN account_records t3 ON t3.uid=l.invitee AND t3.category='task' AND t3.id='3713'
WHERE l.inviter=? ORDER BY l.invitee LIMIT 5
`

type GetInviteProgressRow struct {
	Invitee   model.UserID
	Bound     int64
	Level     int64
	Story     int64
	Purchase1 int64
	Purchase2 int64
	Purchase3 int64
}

func (q *Queries) GetInviteProgress(ctx context.Context, inviter model.UserID) ([]GetInviteProgressRow, error) {
	rows, err := q.query(ctx, q.getInviteProgressStmt, GetInviteProgress, inviter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetInviteProgressRow{}
	for rows.Next() {
		var i GetInviteProgressRow
		if err := rows.Scan(
			&i.Invitee,
			&i.Bound,
			&i.Level,
			&i.Story,
			&i.Purchase1,
			&i.Purchase2,
			&i.Purchase3,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetInviter = `
SELECT inviter,bound FROM invite_links WHERE invitee=?
`

type GetInviterRow struct {
	Inviter model.UserID
	Bound   int64
}

func (q *Queries) GetInviter(ctx context.Context, invitee model.UserID) (GetInviterRow, error) {
	row := q.queryRow(ctx, q.getInviterStmt, GetInviter, invitee)
	var i GetInviterRow
	err := row.Scan(&i.Inviter, &i.Bound)
	return i, err
}

const GetMail = `
SELECT id, owner, title, content, sender, template, created, expires, status, reward_id, items, has_reward FROM mailbox WHERE owner=?1 AND id=?2
`

type GetMailParams struct {
	Owner model.UserID
	ID    model.MailID
}

func (q *Queries) GetMail(ctx context.Context, arg GetMailParams) (Mailbox, error) {
	row := q.queryRow(ctx, q.getMailStmt, GetMail, arg.Owner, arg.ID)
	var i Mailbox
	err := row.Scan(
		&i.ID,
		&i.Owner,
		&i.Title,
		&i.Content,
		&i.Sender,
		&i.Template,
		&i.Created,
		&i.Expires,
		&i.Status,
		&i.RewardID,
		&i.Items,
		&i.HasReward,
	)
	return i, err
}

const GetMigration = `
SELECT completed FROM migrations WHERE name=?1
`

func (q *Queries) GetMigration(ctx context.Context, name string) (int64, error) {
	row := q.queryRow(ctx, q.getMigrationStmt, GetMigration, name)
	var completed int64
	err := row.Scan(&completed)
	return completed, err
}

const GetPublicEquipment = `
SELECT uid,id,category,payload FROM account_records WHERE uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(?1 AS TEXT))) AND category IN ('artifact','highlight') AND coalesce(json_extract(payload,'$.Install'),0)>0
`

type GetPublicEquipmentRow struct {
	UID      model.UserID
	ID       string
	Category model.RecordCategory
	Payload  []byte
}

func (q *Queries) GetPublicEquipment(ctx context.Context, uids string) ([]GetPublicEquipmentRow, error) {
	rows, err := q.query(ctx, q.getPublicEquipmentStmt, GetPublicEquipment, uids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetPublicEquipmentRow{}
	for rows.Next() {
		var i GetPublicEquipmentRow
		if err := rows.Scan(
			&i.UID,
			&i.ID,
			&i.Category,
			&i.Payload,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetPublicHeroes = `
SELECT uid,id,payload FROM account_records WHERE category='hero' AND uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(?1 AS TEXT)))
`

type GetPublicHeroesRow struct {
	UID     model.UserID
	ID      string
	Payload []byte
}

func (q *Queries) GetPublicHeroes(ctx context.Context, uids string) ([]GetPublicHeroesRow, error) {
	rows, err := q.query(ctx, q.getPublicHeroesStmt, GetPublicHeroes, uids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetPublicHeroesRow{}
	for rows.Next() {
		var i GetPublicHeroesRow
		if err := rows.Scan(&i.UID, &i.ID, &i.Payload); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetPublicProfiles = `
SELECT a.uid,a.profile,COALESCE(p.online,0) AS online,COALESCE(p.offline,0) AS offline,(SELECT count(*) FROM account_records h WHERE h.uid=a.uid AND h.category='hero') AS hero_count,COALESCE(m.guild_id,0) AS guild_id,COALESCE(m.role,'') AS guild_role
FROM accounts a LEFT JOIN presence p ON p.uid=a.uid LEFT JOIN guild_members m ON m.uid=a.uid
WHERE a.uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(?1 AS TEXT)))
`

type GetPublicProfilesRow struct {
	UID       model.UserID
	Profile   []byte
	Online    int64
	Offline   int64
	HeroCount int64
	GuildID   model.GuildID
	GuildRole model.GuildRole
}

func (q *Queries) GetPublicProfiles(ctx context.Context, uids string) ([]GetPublicProfilesRow, error) {
	rows, err := q.query(ctx, q.getPublicProfilesStmt, GetPublicProfiles, uids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetPublicProfilesRow{}
	for rows.Next() {
		var i GetPublicProfilesRow
		if err := rows.Scan(
			&i.UID,
			&i.Profile,
			&i.Online,
			&i.Offline,
			&i.HeroCount,
			&i.GuildID,
			&i.GuildRole,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const GetRankEntry = `
SELECT score,achieved FROM rank_entries WHERE uid=? AND mode=? AND rank_id=?
`

type GetRankEntryParams struct {
	UID    model.UserID
	Mode   model.RankMode
	RankID int64
}

type GetRankEntryRow struct {
	Score    int64
	Achieved int64
}

func (q *Queries) GetRankEntry(ctx context.Context, arg GetRankEntryParams) (GetRankEntryRow, error) {
	row := q.queryRow(ctx, q.getRankEntryStmt, GetRankEntry, arg.UID, arg.Mode, arg.RankID)
	var i GetRankEntryRow
	err := row.Scan(&i.Score, &i.Achieved)
	return i, err
}

const GetSocialTimes = `
SELECT search,recommend FROM social_times WHERE uid=?1
`

type GetSocialTimesRow struct {
	Search    int64
	Recommend int64
}

func (q *Queries) GetSocialTimes(ctx context.Context, uid model.UserID) (GetSocialTimesRow, error) {
	row := q.queryRow(ctx, q.getSocialTimesStmt, GetSocialTimes, uid)
	var i GetSocialTimesRow
	err := row.Scan(&i.Search, &i.Recommend)
	return i, err
}

const GetSupportRefresh = `
SELECT refreshed FROM support_refresh WHERE uid=?
`

func (q *Queries) GetSupportRefresh(ctx context.Context, uid model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.getSupportRefreshStmt, GetSupportRefresh, uid)
	var refreshed int64
	err := row.Scan(&refreshed)
	return refreshed, err
}

const GetWorldMembership = `
SELECT channel FROM world_membership WHERE uid=?
`

func (q *Queries) GetWorldMembership(ctx context.Context, uid model.UserID) (string, error) {
	row := q.queryRow(ctx, q.getWorldMembershipStmt, GetWorldMembership, uid)
	var channel string
	err := row.Scan(&channel)
	return channel, err
}

const GiftExists = `
SELECT count(*) FROM friend_gifts WHERE day=?1 AND sender=?2 AND receiver=?3
`

type GiftExistsParams struct {
	Day      int64
	Sender   model.UserID
	Receiver model.UserID
}

func (q *Queries) GiftExists(ctx context.Context, arg GiftExistsParams) (int64, error) {
	row := q.queryRow(ctx, q.giftExistsStmt, GiftExists, arg.Day, arg.Sender, arg.Receiver)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const GuildAdminCount = `
SELECT count(*) FROM guild_members WHERE guild_id=?1 AND role='manager'
`

func (q *Queries) GuildAdminCount(ctx context.Context, guildID model.GuildID) (int64, error) {
	row := q.queryRow(ctx, q.guildAdminCountStmt, GuildAdminCount, guildID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const GuildNameExists = `
SELECT count(*) FROM guilds WHERE name=?1 COLLATE NOCASE
`

func (q *Queries) GuildNameExists(ctx context.Context, name string) (int64, error) {
	row := q.queryRow(ctx, q.guildNameExistsStmt, GuildNameExists, name)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const InsertBlock = `
INSERT INTO blocks(owner,peer,created) VALUES (?1,?2,?3) ON CONFLICT(owner,peer) DO NOTHING
`

type InsertBlockParams struct {
	Owner   model.UserID
	Peer    model.UserID
	Created int64
}

func (q *Queries) InsertBlock(ctx context.Context, arg InsertBlockParams) error {
	_, err := q.exec(ctx, q.insertBlockStmt, InsertBlock, arg.Owner, arg.Peer, arg.Created)
	return err
}

const InsertFriendPair = `
INSERT INTO friends(owner,peer,added) VALUES (?1,?2,?3),(?2,?1,?3) ON CONFLICT(owner,peer) DO NOTHING
`

type InsertFriendPairParams struct {
	Owner model.UserID
	Peer  model.UserID
	Added int64
}

func (q *Queries) InsertFriendPair(ctx context.Context, arg InsertFriendPairParams) error {
	_, err := q.exec(ctx, q.insertFriendPairStmt, InsertFriendPair, arg.Owner, arg.Peer, arg.Added)
	return err
}

const InsertFriendRequest = `
INSERT INTO friend_requests(receiver,sender,source,created) VALUES (?1,?2,?3,?4) ON CONFLICT(receiver,sender) DO NOTHING
`

type InsertFriendRequestParams struct {
	Receiver model.UserID
	Sender   model.UserID
	Source   model.FriendSource
	Created  int64
}

func (q *Queries) InsertFriendRequest(ctx context.Context, arg InsertFriendRequestParams) error {
	_, err := q.exec(ctx, q.insertFriendRequestStmt, InsertFriendRequest,
		arg.Receiver,
		arg.Sender,
		arg.Source,
		arg.Created,
	)
	return err
}

const InsertGift = `
INSERT INTO friend_gifts(day,sender,receiver) VALUES (?1,?2,?3)
`

type InsertGiftParams struct {
	Day      int64
	Sender   model.UserID
	Receiver model.UserID
}

func (q *Queries) InsertGift(ctx context.Context, arg InsertGiftParams) error {
	_, err := q.exec(ctx, q.insertGiftStmt, InsertGift, arg.Day, arg.Sender, arg.Receiver)
	return err
}

const InsertGuildApplication = `
INSERT INTO guild_applications(guild_id,applicant,created) VALUES (?1,?2,?3) ON CONFLICT(guild_id,applicant) DO UPDATE SET created=excluded.created
`

type InsertGuildApplicationParams struct {
	GuildID   model.GuildID
	Applicant model.UserID
	Created   int64
}

func (q *Queries) InsertGuildApplication(ctx context.Context, arg InsertGuildApplicationParams) error {
	_, err := q.exec(ctx, q.insertGuildApplicationStmt, InsertGuildApplication, arg.GuildID, arg.Applicant, arg.Created)
	return err
}

const InsertGuildInvite = `
INSERT INTO guild_invites(guild_id,player,inviter,created) VALUES (?1,?2,?3,?4) ON CONFLICT(guild_id,player) DO UPDATE SET inviter=excluded.inviter,created=excluded.created
`

type InsertGuildInviteParams struct {
	GuildID model.GuildID
	Player  model.UserID
	Inviter model.UserID
	Created int64
}

func (q *Queries) InsertGuildInvite(ctx context.Context, arg InsertGuildInviteParams) error {
	_, err := q.exec(ctx, q.insertGuildInviteStmt, InsertGuildInvite,
		arg.GuildID,
		arg.Player,
		arg.Inviter,
		arg.Created,
	)
	return err
}

const InsertGuildMember = `
INSERT INTO guild_members(uid,guild_id,role,joined) VALUES (?1,?2,?3,?4)
`

type InsertGuildMemberParams struct {
	UID     model.UserID
	GuildID model.GuildID
	Role    model.GuildRole
	Joined  int64
}

func (q *Queries) InsertGuildMember(ctx context.Context, arg InsertGuildMemberParams) error {
	_, err := q.exec(ctx, q.insertGuildMemberStmt, InsertGuildMember,
		arg.UID,
		arg.GuildID,
		arg.Role,
		arg.Joined,
	)
	return err
}

const InsertGuildMessage = `
INSERT INTO guild_chat(guild_id,sender,kind,content,created) VALUES (?1,?2,?3,?4,?5) RETURNING id
`

type InsertGuildMessageParams struct {
	GuildID model.GuildID
	Sender  model.UserID
	Kind    int64
	Content string
	Created int64
}

func (q *Queries) InsertGuildMessage(ctx context.Context, arg InsertGuildMessageParams) (int64, error) {
	row := q.queryRow(ctx, q.insertGuildMessageStmt, InsertGuildMessage,
		arg.GuildID,
		arg.Sender,
		arg.Kind,
		arg.Content,
		arg.Created,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const InsertGuildNews = `
INSERT INTO guild_news(guild_id,kind,actor,subject,value,created) VALUES (?1,?2,?3,?4,?5,?6) RETURNING id
`

type InsertGuildNewsParams struct {
	GuildID model.GuildID
	Kind    int64
	Actor   model.UserID
	Subject model.UserID
	Value   int64
	Created int64
}

func (q *Queries) InsertGuildNews(ctx context.Context, arg InsertGuildNewsParams) (int64, error) {
	row := q.queryRow(ctx, q.insertGuildNewsStmt, InsertGuildNews,
		arg.GuildID,
		arg.Kind,
		arg.Actor,
		arg.Subject,
		arg.Value,
		arg.Created,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const InsertGuildReport = `
INSERT INTO guild_reports(reporter,guild_id,reason,content,created) VALUES (?1,?2,?3,?4,?5)
`

type InsertGuildReportParams struct {
	Reporter model.UserID
	GuildID  model.GuildID
	Reason   string
	Content  string
	Created  int64
}

func (q *Queries) InsertGuildReport(ctx context.Context, arg InsertGuildReportParams) error {
	_, err := q.exec(ctx, q.insertGuildReportStmt, InsertGuildReport,
		arg.Reporter,
		arg.GuildID,
		arg.Reason,
		arg.Content,
		arg.Created,
	)
	return err
}

const InsertInvite = `
INSERT INTO invite_links(invitee,inviter,bound) VALUES (?,?,?)
`

type InsertInviteParams struct {
	Invitee model.UserID
	Inviter model.UserID
	Bound   int64
}

func (q *Queries) InsertInvite(ctx context.Context, arg InsertInviteParams) error {
	_, err := q.exec(ctx, q.insertInviteStmt, InsertInvite, arg.Invitee, arg.Inviter, arg.Bound)
	return err
}

const InsertPrivateMessage = `
INSERT INTO chat_messages(room,channel,sender,receiver,kind,content,created) VALUES (?1,1,?2,?3,?4,?5,?6) RETURNING id
`

type InsertPrivateMessageParams struct {
	Room     string
	Sender   model.UserID
	Receiver model.UserID
	Kind     int64
	Content  string
	Created  int64
}

func (q *Queries) InsertPrivateMessage(ctx context.Context, arg InsertPrivateMessageParams) (int64, error) {
	row := q.queryRow(ctx, q.insertPrivateMessageStmt, InsertPrivateMessage,
		arg.Room,
		arg.Sender,
		arg.Receiver,
		arg.Kind,
		arg.Content,
		arg.Created,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const ListAccountUIDs = `
SELECT uid FROM accounts ORDER BY uid
`

func (q *Queries) ListAccountUIDs(ctx context.Context) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listAccountUIDsStmt, ListAccountUIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListApplications = `
SELECT sender,source,created FROM friend_requests WHERE receiver=?1
`

type ListApplicationsRow struct {
	Sender  model.UserID
	Source  model.FriendSource
	Created int64
}

func (q *Queries) ListApplications(ctx context.Context, receiver model.UserID) ([]ListApplicationsRow, error) {
	rows, err := q.query(ctx, q.listApplicationsStmt, ListApplications, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListApplicationsRow{}
	for rows.Next() {
		var i ListApplicationsRow
		if err := rows.Scan(&i.Sender, &i.Source, &i.Created); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListAsyncCandidates = `
SELECT a.uid FROM accounts a WHERE a.uid<>?1 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=?1 AND b.peer=a.uid) ORDER BY a.last_login DESC,a.uid LIMIT 30
`

func (q *Queries) ListAsyncCandidates(ctx context.Context, uid model.UserID) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listAsyncCandidatesStmt, ListAsyncCandidates, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListAsyncRecords = `
SELECT record FROM async_records WHERE (attacker=?1 OR defender=?1) AND friend=?2 ORDER BY id DESC LIMIT 50
`

type ListAsyncRecordsParams struct {
	UID    model.UserID
	Friend int64
}

func (q *Queries) ListAsyncRecords(ctx context.Context, arg ListAsyncRecordsParams) ([]string, error) {
	rows, err := q.query(ctx, q.listAsyncRecordsStmt, ListAsyncRecords, arg.UID, arg.Friend)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var record string
		if err := rows.Scan(&record); err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListBlocks = `
SELECT peer FROM blocks WHERE owner=?1 ORDER BY peer
`

func (q *Queries) ListBlocks(ctx context.Context, owner model.UserID) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listBlocksStmt, ListBlocks, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var peer model.UserID
		if err := rows.Scan(&peer); err != nil {
			return nil, err
		}
		items = append(items, peer)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListCommunityMessages = `
SELECT uid,content,updated FROM community_messages WHERE scope=?1 AND subject=?2 ORDER BY updated DESC,uid LIMIT ?3
`

type ListCommunityMessagesParams struct {
	Scope   model.MessageScope
	Subject int64
	MaxRows int64
}

type ListCommunityMessagesRow struct {
	UID     model.UserID
	Content string
	Updated int64
}

func (q *Queries) ListCommunityMessages(ctx context.Context, arg ListCommunityMessagesParams) ([]ListCommunityMessagesRow, error) {
	rows, err := q.query(ctx, q.listCommunityMessagesStmt, ListCommunityMessages, arg.Scope, arg.Subject, arg.MaxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListCommunityMessagesRow{}
	for rows.Next() {
		var i ListCommunityMessagesRow
		if err := rows.Scan(&i.UID, &i.Content, &i.Updated); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListFriendRanks = `
SELECT e.uid,e.score,e.achieved FROM rank_entries e JOIN friends f ON f.peer=e.uid WHERE f.owner=? AND e.mode=? AND e.rank_id=? ORDER BY e.score DESC,e.achieved,e.uid
`

type ListFriendRanksParams struct {
	Owner  model.UserID
	Mode   model.RankMode
	RankID int64
}

type ListFriendRanksRow struct {
	UID      model.UserID
	Score    int64
	Achieved int64
}

func (q *Queries) ListFriendRanks(ctx context.Context, arg ListFriendRanksParams) ([]ListFriendRanksRow, error) {
	rows, err := q.query(ctx, q.listFriendRanksStmt, ListFriendRanks, arg.Owner, arg.Mode, arg.RankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListFriendRanksRow{}
	for rows.Next() {
		var i ListFriendRanksRow
		if err := rows.Scan(&i.UID, &i.Score, &i.Achieved); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListFriends = `
SELECT peer,nickname,added FROM friends WHERE owner=?1
`

type ListFriendsRow struct {
	Peer     model.UserID
	Nickname string
	Added    int64
}

func (q *Queries) ListFriends(ctx context.Context, owner model.UserID) ([]ListFriendsRow, error) {
	rows, err := q.query(ctx, q.listFriendsStmt, ListFriends, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListFriendsRow{}
	for rows.Next() {
		var i ListFriendsRow
		if err := rows.Scan(&i.Peer, &i.Nickname, &i.Added); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildApplications = `
SELECT applicant,created FROM guild_applications WHERE guild_id=?1 AND created>?2
`

type ListGuildApplicationsParams struct {
	GuildID model.GuildID
	After   int64
}

type ListGuildApplicationsRow struct {
	Applicant model.UserID
	Created   int64
}

func (q *Queries) ListGuildApplications(ctx context.Context, arg ListGuildApplicationsParams) ([]ListGuildApplicationsRow, error) {
	rows, err := q.query(ctx, q.listGuildApplicationsStmt, ListGuildApplications, arg.GuildID, arg.After)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildApplicationsRow{}
	for rows.Next() {
		var i ListGuildApplicationsRow
		if err := rows.Scan(&i.Applicant, &i.Created); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildContributions = `
SELECT uid,week,score FROM guild_contributions WHERE guild_id=?1 AND week>=?2 AND week<?3 ORDER BY week,uid
`

type ListGuildContributionsParams struct {
	GuildID  model.GuildID
	FromWeek int64
	ToWeek   int64
}

type ListGuildContributionsRow struct {
	UID   model.UserID
	Week  int64
	Score int64
}

func (q *Queries) ListGuildContributions(ctx context.Context, arg ListGuildContributionsParams) ([]ListGuildContributionsRow, error) {
	rows, err := q.query(ctx, q.listGuildContributionsStmt, ListGuildContributions, arg.GuildID, arg.FromWeek, arg.ToWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildContributionsRow{}
	for rows.Next() {
		var i ListGuildContributionsRow
		if err := rows.Scan(&i.UID, &i.Week, &i.Score); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildInvites = `
SELECT player,created FROM guild_invites WHERE guild_id=?1 AND created>?2
`

type ListGuildInvitesParams struct {
	GuildID model.GuildID
	After   int64
}

type ListGuildInvitesRow struct {
	Player  model.UserID
	Created int64
}

func (q *Queries) ListGuildInvites(ctx context.Context, arg ListGuildInvitesParams) ([]ListGuildInvitesRow, error) {
	rows, err := q.query(ctx, q.listGuildInvitesStmt, ListGuildInvites, arg.GuildID, arg.After)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildInvitesRow{}
	for rows.Next() {
		var i ListGuildInvitesRow
		if err := rows.Scan(&i.Player, &i.Created); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildMembers = `
SELECT m.uid,m.role,m.joined,m.score,m.history_score,COALESCE(p.online,0) AS online,COALESCE(p.offline,0) AS offline FROM guild_members m LEFT JOIN presence p ON m.uid=p.uid WHERE m.guild_id=?1 ORDER BY m.uid
`

type ListGuildMembersRow struct {
	UID          model.UserID
	Role         model.GuildRole
	Joined       int64
	Score        int64
	HistoryScore int64
	Online       int64
	Offline      int64
}

func (q *Queries) ListGuildMembers(ctx context.Context, guildID model.GuildID) ([]ListGuildMembersRow, error) {
	rows, err := q.query(ctx, q.listGuildMembersStmt, ListGuildMembers, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildMembersRow{}
	for rows.Next() {
		var i ListGuildMembersRow
		if err := rows.Scan(
			&i.UID,
			&i.Role,
			&i.Joined,
			&i.Score,
			&i.HistoryScore,
			&i.Online,
			&i.Offline,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildMessages = `
SELECT id,sender,kind,content,created FROM guild_chat WHERE guild_id=?1 ORDER BY id DESC LIMIT ?2
`

type ListGuildMessagesParams struct {
	GuildID model.GuildID
	MaxRows int64
}

type ListGuildMessagesRow struct {
	ID      int64
	Sender  model.UserID
	Kind    int64
	Content string
	Created int64
}

func (q *Queries) ListGuildMessages(ctx context.Context, arg ListGuildMessagesParams) ([]ListGuildMessagesRow, error) {
	rows, err := q.query(ctx, q.listGuildMessagesStmt, ListGuildMessages, arg.GuildID, arg.MaxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildMessagesRow{}
	for rows.Next() {
		var i ListGuildMessagesRow
		if err := rows.Scan(
			&i.ID,
			&i.Sender,
			&i.Kind,
			&i.Content,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildNews = `
SELECT id, guild_id, kind, actor, subject, value, created FROM guild_news WHERE guild_id=?1 ORDER BY id DESC LIMIT 100
`

func (q *Queries) ListGuildNews(ctx context.Context, guildID model.GuildID) ([]GuildNews, error) {
	rows, err := q.query(ctx, q.listGuildNewsStmt, ListGuildNews, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GuildNews{}
	for rows.Next() {
		var i GuildNews
		if err := rows.Scan(
			&i.ID,
			&i.GuildID,
			&i.Kind,
			&i.Actor,
			&i.Subject,
			&i.Value,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildRanks = `
SELECT e.uid,e.score,e.achieved FROM rank_entries e JOIN guild_members m ON m.uid=e.uid WHERE m.guild_id=(SELECT own.guild_id FROM guild_members own WHERE own.uid=?) AND e.mode=? AND e.rank_id=? ORDER BY e.score DESC,e.achieved,e.uid
`

type ListGuildRanksParams struct {
	UID    model.UserID
	Mode   model.RankMode
	RankID int64
}

type ListGuildRanksRow struct {
	UID      model.UserID
	Score    int64
	Achieved int64
}

func (q *Queries) ListGuildRanks(ctx context.Context, arg ListGuildRanksParams) ([]ListGuildRanksRow, error) {
	rows, err := q.query(ctx, q.listGuildRanksStmt, ListGuildRanks, arg.UID, arg.Mode, arg.RankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildRanksRow{}
	for rows.Next() {
		var i ListGuildRanksRow
		if err := rows.Scan(&i.UID, &i.Score, &i.Achieved); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildWarHistory = `
SELECT record FROM guild_war_history WHERE guild_id=? AND war=? AND area=? AND group_id=? ORDER BY duration,id LIMIT 50
`

type ListGuildWarHistoryParams struct {
	GuildID model.GuildID
	War     int64
	Area    int64
	GroupID int64
}

func (q *Queries) ListGuildWarHistory(ctx context.Context, arg ListGuildWarHistoryParams) ([][]byte, error) {
	rows, err := q.query(ctx, q.listGuildWarHistoryStmt, ListGuildWarHistory,
		arg.GuildID,
		arg.War,
		arg.Area,
		arg.GroupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := [][]byte{}
	for rows.Next() {
		var record []byte
		if err := rows.Scan(&record); err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListGuildWarProgress = `
SELECT uid, area, score, assist, score_time, assist_time FROM guild_war_progress WHERE guild_id=? AND war=?
`

type ListGuildWarProgressParams struct {
	GuildID model.GuildID
	War     int64
}

type ListGuildWarProgressRow struct {
	UID        model.UserID
	Area       int64
	Score      int64
	Assist     int64
	ScoreTime  int64
	AssistTime int64
}

func (q *Queries) ListGuildWarProgress(ctx context.Context, arg ListGuildWarProgressParams) ([]ListGuildWarProgressRow, error) {
	rows, err := q.query(ctx, q.listGuildWarProgressStmt, ListGuildWarProgress, arg.GuildID, arg.War)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListGuildWarProgressRow{}
	for rows.Next() {
		var i ListGuildWarProgressRow
		if err := rows.Scan(
			&i.UID,
			&i.Area,
			&i.Score,
			&i.Assist,
			&i.ScoreTime,
			&i.AssistTime,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListInvitees = `
SELECT invitee FROM invite_links WHERE inviter=? ORDER BY invitee
`

func (q *Queries) ListInvitees(ctx context.Context, inviter model.UserID) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listInviteesStmt, ListInvitees, inviter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var invitee model.UserID
		if err := rows.Scan(&invitee); err != nil {
			return nil, err
		}
		items = append(items, invitee)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListMailbox = `
SELECT id, owner, title, content, sender, template, created, expires, status, reward_id, items, has_reward FROM mailbox WHERE owner=?1 AND (expires=0 OR expires>?2) ORDER BY id DESC LIMIT ?3
`

type ListMailboxParams struct {
	Owner   model.UserID
	Now     int64
	MaxRows int64
}

func (q *Queries) ListMailbox(ctx context.Context, arg ListMailboxParams) ([]Mailbox, error) {
	rows, err := q.query(ctx, q.listMailboxStmt, ListMailbox, arg.Owner, arg.Now, arg.MaxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Mailbox{}
	for rows.Next() {
		var i Mailbox
		if err := rows.Scan(
			&i.ID,
			&i.Owner,
			&i.Title,
			&i.Content,
			&i.Sender,
			&i.Template,
			&i.Created,
			&i.Expires,
			&i.Status,
			&i.RewardID,
			&i.Items,
			&i.HasReward,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListPendingMessages = `
SELECT id,sender,receiver,kind,content,created FROM chat_messages c WHERE receiver=?1 AND channel=1 AND acknowledged=0 AND EXISTS(SELECT 1 FROM friends f WHERE f.owner=c.receiver AND f.peer=c.sender) ORDER BY id DESC LIMIT ?2
`

type ListPendingMessagesParams struct {
	Receiver model.UserID
	MaxRows  int64
}

type ListPendingMessagesRow struct {
	ID       int64
	Sender   model.UserID
	Receiver model.UserID
	Kind     int64
	Content  string
	Created  int64
}

func (q *Queries) ListPendingMessages(ctx context.Context, arg ListPendingMessagesParams) ([]ListPendingMessagesRow, error) {
	rows, err := q.query(ctx, q.listPendingMessagesStmt, ListPendingMessages, arg.Receiver, arg.MaxRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListPendingMessagesRow{}
	for rows.Next() {
		var i ListPendingMessagesRow
		if err := rows.Scan(
			&i.ID,
			&i.Sender,
			&i.Receiver,
			&i.Kind,
			&i.Content,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListPendingNetworkAccounts = `
SELECT uid, CAST(json_extract(profile,'$.network') AS BLOB) AS network FROM accounts WHERE json_type(profile,'$.network.pending')='object'
`

type ListPendingNetworkAccountsRow struct {
	UID     model.UserID
	Network []byte
}

func (q *Queries) ListPendingNetworkAccounts(ctx context.Context) ([]ListPendingNetworkAccountsRow, error) {
	rows, err := q.query(ctx, q.listPendingNetworkAccountsStmt, ListPendingNetworkAccounts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListPendingNetworkAccountsRow{}
	for rows.Next() {
		var i ListPendingNetworkAccountsRow
		if err := rows.Scan(&i.UID, &i.Network); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListPlayerGuildApplications = `
SELECT guild_id,created FROM guild_applications WHERE applicant=?1 AND created>?2
`

type ListPlayerGuildApplicationsParams struct {
	Applicant model.UserID
	After     int64
}

type ListPlayerGuildApplicationsRow struct {
	GuildID model.GuildID
	Created int64
}

func (q *Queries) ListPlayerGuildApplications(ctx context.Context, arg ListPlayerGuildApplicationsParams) ([]ListPlayerGuildApplicationsRow, error) {
	rows, err := q.query(ctx, q.listPlayerGuildApplicationsStmt, ListPlayerGuildApplications, arg.Applicant, arg.After)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListPlayerGuildApplicationsRow{}
	for rows.Next() {
		var i ListPlayerGuildApplicationsRow
		if err := rows.Scan(&i.GuildID, &i.Created); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListPlayerGuildInvites = `
SELECT guild_id,created FROM guild_invites WHERE player=?1 AND created>?2
`

type ListPlayerGuildInvitesParams struct {
	Player model.UserID
	After  int64
}

type ListPlayerGuildInvitesRow struct {
	GuildID model.GuildID
	Created int64
}

func (q *Queries) ListPlayerGuildInvites(ctx context.Context, arg ListPlayerGuildInvitesParams) ([]ListPlayerGuildInvitesRow, error) {
	rows, err := q.query(ctx, q.listPlayerGuildInvitesStmt, ListPlayerGuildInvites, arg.Player, arg.After)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListPlayerGuildInvitesRow{}
	for rows.Next() {
		var i ListPlayerGuildInvitesRow
		if err := rows.Scan(&i.GuildID, &i.Created); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListPlayerRanks = `
SELECT mode,rank_id,score,achieved FROM rank_entries WHERE uid=? ORDER BY mode,rank_id
`

type ListPlayerRanksRow struct {
	Mode     model.RankMode
	RankID   int64
	Score    int64
	Achieved int64
}

func (q *Queries) ListPlayerRanks(ctx context.Context, uid model.UserID) ([]ListPlayerRanksRow, error) {
	rows, err := q.query(ctx, q.listPlayerRanksStmt, ListPlayerRanks, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListPlayerRanksRow{}
	for rows.Next() {
		var i ListPlayerRanksRow
		if err := rows.Scan(
			&i.Mode,
			&i.RankID,
			&i.Score,
			&i.Achieved,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListReceivedGifts = `
SELECT sender,claimed FROM friend_gifts WHERE receiver=?1 AND day=?2
`

type ListReceivedGiftsParams struct {
	Receiver model.UserID
	Day      int64
}

type ListReceivedGiftsRow struct {
	Sender  model.UserID
	Claimed int64
}

func (q *Queries) ListReceivedGifts(ctx context.Context, arg ListReceivedGiftsParams) ([]ListReceivedGiftsRow, error) {
	rows, err := q.query(ctx, q.listReceivedGiftsStmt, ListReceivedGifts, arg.Receiver, arg.Day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListReceivedGiftsRow{}
	for rows.Next() {
		var i ListReceivedGiftsRow
		if err := rows.Scan(&i.Sender, &i.Claimed); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListRecords = `
SELECT category,id,payload FROM account_records WHERE uid=?1
`

type ListRecordsRow struct {
	Category model.RecordCategory
	ID       string
	Payload  []byte
}

func (q *Queries) ListRecords(ctx context.Context, uid model.UserID) ([]ListRecordsRow, error) {
	rows, err := q.query(ctx, q.listRecordsStmt, ListRecords, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListRecordsRow{}
	for rows.Next() {
		var i ListRecordsRow
		if err := rows.Scan(&i.Category, &i.ID, &i.Payload); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListSentGifts = `
SELECT receiver FROM friend_gifts WHERE sender=?1 AND day=?2
`

type ListSentGiftsParams struct {
	Sender model.UserID
	Day    int64
}

func (q *Queries) ListSentGifts(ctx context.Context, arg ListSentGiftsParams) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listSentGiftsStmt, ListSentGifts, arg.Sender, arg.Day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var receiver model.UserID
		if err := rows.Scan(&receiver); err != nil {
			return nil, err
		}
		items = append(items, receiver)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListSupportCandidates = `
SELECT a.uid FROM accounts a WHERE a.uid<>?1 AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.owner=?1 AND b.peer=a.uid) ORDER BY EXISTS(SELECT 1 FROM friends f WHERE f.owner=?1 AND f.peer=a.uid) DESC,a.last_login DESC,a.uid LIMIT 30
`

func (q *Queries) ListSupportCandidates(ctx context.Context, uid model.UserID) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listSupportCandidatesStmt, ListSupportCandidates, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListSupportUsage = `
SELECT borrower,hero,uses,claimed,day FROM support_usage WHERE owner=? ORDER BY day DESC,borrower,hero
`

type ListSupportUsageRow struct {
	Borrower model.UserID
	Hero     model.HeroID
	Uses     int64
	Claimed  int64
	Day      int64
}

func (q *Queries) ListSupportUsage(ctx context.Context, owner model.UserID) ([]ListSupportUsageRow, error) {
	rows, err := q.query(ctx, q.listSupportUsageStmt, ListSupportUsage, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListSupportUsageRow{}
	for rows.Next() {
		var i ListSupportUsageRow
		if err := rows.Scan(
			&i.Borrower,
			&i.Hero,
			&i.Uses,
			&i.Claimed,
			&i.Day,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListWorldMembers = `
SELECT uid FROM world_membership WHERE channel=? ORDER BY uid
`

func (q *Queries) ListWorldMembers(ctx context.Context, channel string) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.listWorldMembersStmt, ListWorldMembers, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const ListWorldMessages = `
SELECT id,sender,kind,content,created FROM world_messages WHERE channel=? ORDER BY id DESC LIMIT 100
`

type ListWorldMessagesRow struct {
	ID      int64
	Sender  model.UserID
	Kind    int64
	Content string
	Created int64
}

func (q *Queries) ListWorldMessages(ctx context.Context, channel string) ([]ListWorldMessagesRow, error) {
	rows, err := q.query(ctx, q.listWorldMessagesStmt, ListWorldMessages, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListWorldMessagesRow{}
	for rows.Next() {
		var i ListWorldMessagesRow
		if err := rows.Scan(
			&i.ID,
			&i.Sender,
			&i.Kind,
			&i.Content,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const MailboxCount = `
SELECT count(*) FROM mailbox WHERE owner=?1
`

func (q *Queries) MailboxCount(ctx context.Context, owner model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.mailboxCountStmt, MailboxCount, owner)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const MarkGiftClaimed = `
UPDATE friend_gifts SET claimed=1 WHERE day=?1 AND receiver=?2 AND sender=?3
`

type MarkGiftClaimedParams struct {
	Day      int64
	Receiver model.UserID
	Sender   model.UserID
}

func (q *Queries) MarkGiftClaimed(ctx context.Context, arg MarkGiftClaimedParams) error {
	_, err := q.exec(ctx, q.markGiftClaimedStmt, MarkGiftClaimed, arg.Day, arg.Receiver, arg.Sender)
	return err
}

const MigrateAccountUID = `
UPDATE accounts SET uid=?1,profile=json_set(profile,'$.uid',?1) WHERE uid=?2
`

type MigrateAccountUIDParams struct {
	NewUID model.UserID
	OldUID model.UserID
}

func (q *Queries) MigrateAccountUID(ctx context.Context, arg MigrateAccountUIDParams) error {
	_, err := q.exec(ctx, q.migrateAccountUIDStmt, MigrateAccountUID, arg.NewUID, arg.OldUID)
	return err
}

const NextAccountUID = `
SELECT max(COALESCE(max(uid),0),100000)+1 FROM accounts
`

func (q *Queries) NextAccountUID(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.nextAccountUIDStmt, NextAccountUID)
	var column_1 int64
	err := row.Scan(&column_1)
	return column_1, err
}

const OpenMail = `
UPDATE mailbox SET status=2 WHERE owner=?1 AND id=?2 AND status=1 AND (expires=0 OR expires>?3)
`

type OpenMailParams struct {
	Owner model.UserID
	ID    model.MailID
	Now   int64
}

func (q *Queries) OpenMail(ctx context.Context, arg OpenMailParams) error {
	_, err := q.exec(ctx, q.openMailStmt, OpenMail, arg.Owner, arg.ID, arg.Now)
	return err
}

const PlayerLevel = `
SELECT level FROM accounts WHERE uid=?1
`

func (q *Queries) PlayerLevel(ctx context.Context, uid model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.playerLevelStmt, PlayerLevel, uid)
	var level int64
	err := row.Scan(&level)
	return level, err
}

const PruneWorldMessages = `
DELETE FROM world_messages WHERE world_messages.channel=?1 AND id < coalesce((SELECT w.id FROM world_messages w WHERE w.channel=?1 ORDER BY id DESC LIMIT 1 OFFSET 199),0)
`

func (q *Queries) PruneWorldMessages(ctx context.Context, channel string) error {
	_, err := q.exec(ctx, q.pruneWorldMessagesStmt, PruneWorldMessages, channel)
	return err
}

const RankBoardSize = `
SELECT count(*) FROM rank_entries WHERE mode=? AND rank_id=?
`

type RankBoardSizeParams struct {
	Mode   model.RankMode
	RankID int64
}

func (q *Queries) RankBoardSize(ctx context.Context, arg RankBoardSizeParams) (int64, error) {
	row := q.queryRow(ctx, q.rankBoardSizeStmt, RankBoardSize, arg.Mode, arg.RankID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const RankHigherCount = `
SELECT count(*) FROM rank_entries WHERE mode=?1 AND rank_id=?2 AND (score>?3 OR (score=?3 AND achieved<?4) OR (score=?3 AND achieved=?4 AND uid<?5))
`

type RankHigherCountParams struct {
	Mode     model.RankMode
	RankID   int64
	Score    int64
	Achieved int64
	UID      model.UserID
}

func (q *Queries) RankHigherCount(ctx context.Context, arg RankHigherCountParams) (int64, error) {
	row := q.queryRow(ctx, q.rankHigherCountStmt, RankHigherCount,
		arg.Mode,
		arg.RankID,
		arg.Score,
		arg.Achieved,
		arg.UID,
	)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const ReadClockOffset = `
SELECT offset_seconds FROM server_clock WHERE singleton=1
`

func (q *Queries) ReadClockOffset(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.readClockOffsetStmt, ReadClockOffset)
	var offset_seconds int64
	err := row.Scan(&offset_seconds)
	return offset_seconds, err
}

const ReceivedGiftCount = `
SELECT count(*) FROM friend_gifts WHERE day=?1 AND receiver=?2 AND claimed=1
`

type ReceivedGiftCountParams struct {
	Day      int64
	Receiver model.UserID
}

func (q *Queries) ReceivedGiftCount(ctx context.Context, arg ReceivedGiftCountParams) (int64, error) {
	row := q.queryRow(ctx, q.receivedGiftCountStmt, ReceivedGiftCount, arg.Day, arg.Receiver)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const RecommendGuilds = `
SELECT id FROM guilds WHERE minimum_level<=?1 AND member_count<?2 ORDER BY member_count DESC,id LIMIT 20
`

type RecommendGuildsParams struct {
	Level    int64
	Capacity int64
}

func (q *Queries) RecommendGuilds(ctx context.Context, arg RecommendGuildsParams) ([]model.GuildID, error) {
	rows, err := q.query(ctx, q.recommendGuildsStmt, RecommendGuilds, arg.Level, arg.Capacity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.GuildID{}
	for rows.Next() {
		var id model.GuildID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const RecommendPlayers = `
SELECT a.uid FROM accounts a WHERE a.uid!=?1 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.owner=?1 AND b.peer=a.uid) OR (b.owner=a.uid AND b.peer=?1)) AND NOT EXISTS(SELECT 1 FROM friends f WHERE f.owner=?1 AND f.peer=a.uid) AND NOT EXISTS(SELECT 1 FROM friend_requests q WHERE (q.receiver=a.uid AND q.sender=?1) OR (q.receiver=?1 AND q.sender=a.uid)) ORDER BY a.last_login DESC,a.uid LIMIT 20
`

func (q *Queries) RecommendPlayers(ctx context.Context, self model.UserID) ([]model.UserID, error) {
	rows, err := q.query(ctx, q.recommendPlayersStmt, RecommendPlayers, self)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.UserID{}
	for rows.Next() {
		var uid model.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		items = append(items, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const RequestCount = `
SELECT count(*) FROM friend_requests WHERE receiver=?1
`

func (q *Queries) RequestCount(ctx context.Context, receiver model.UserID) (int64, error) {
	row := q.queryRow(ctx, q.requestCountStmt, RequestCount, receiver)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const ResolveLegacyUID = `
SELECT new_uid FROM uid_history WHERE old_uid=?1
`

func (q *Queries) ResolveLegacyUID(ctx context.Context, oldUid model.UserID) (model.UserID, error) {
	row := q.queryRow(ctx, q.resolveLegacyUIDStmt, ResolveLegacyUID, oldUid)
	var new_uid model.UserID
	err := row.Scan(&new_uid)
	return new_uid, err
}

const SaveAsyncRecord = `
INSERT INTO async_records(attacker,defender,opponent,friend,created,record) VALUES (?,?,?,?,?,?)
`

type SaveAsyncRecordParams struct {
	Attacker model.UserID
	Defender sql.NullInt64
	Opponent int64
	Friend   int64
	Created  int64
	Record   string
}

func (q *Queries) SaveAsyncRecord(ctx context.Context, arg SaveAsyncRecordParams) error {
	_, err := q.exec(ctx, q.saveAsyncRecordStmt, SaveAsyncRecord,
		arg.Attacker,
		arg.Defender,
		arg.Opponent,
		arg.Friend,
		arg.Created,
		arg.Record,
	)
	return err
}

const SaveClockOffset = `
INSERT INTO server_clock(singleton,offset_seconds) VALUES (1,?) ON CONFLICT(singleton) DO UPDATE SET offset_seconds=excluded.offset_seconds
`

func (q *Queries) SaveClockOffset(ctx context.Context, offsetSeconds int64) error {
	_, err := q.exec(ctx, q.saveClockOffsetStmt, SaveClockOffset, offsetSeconds)
	return err
}

const SaveCommunityMessage = `
INSERT INTO community_messages(scope,subject,uid,content,updated) VALUES (?1,?2,?3,?4,?5) ON CONFLICT(scope,subject,uid) DO UPDATE SET content=excluded.content,updated=excluded.updated
`

type SaveCommunityMessageParams struct {
	Scope   model.MessageScope
	Subject int64
	UID     model.UserID
	Content string
	Updated int64
}

func (q *Queries) SaveCommunityMessage(ctx context.Context, arg SaveCommunityMessageParams) error {
	_, err := q.exec(ctx, q.saveCommunityMessageStmt, SaveCommunityMessage,
		arg.Scope,
		arg.Subject,
		arg.UID,
		arg.Content,
		arg.Updated,
	)
	return err
}

const SaveGuildWar = `
INSERT INTO guild_wars(guild_id,war,state) VALUES (?,?,?) ON CONFLICT(guild_id,war) DO UPDATE SET state=excluded.state
`

type SaveGuildWarParams struct {
	GuildID model.GuildID
	War     int64
	State   []byte
}

func (q *Queries) SaveGuildWar(ctx context.Context, arg SaveGuildWarParams) error {
	_, err := q.exec(ctx, q.saveGuildWarStmt, SaveGuildWar, arg.GuildID, arg.War, arg.State)
	return err
}

const SaveGuildWarHistory = `
INSERT INTO guild_war_history(guild_id,war,area,group_id,uid,duration,record) VALUES (?,?,?,?,?,?,?)
`

type SaveGuildWarHistoryParams struct {
	GuildID  model.GuildID
	War      int64
	Area     int64
	GroupID  int64
	UID      model.UserID
	Duration int64
	Record   []byte
}

func (q *Queries) SaveGuildWarHistory(ctx context.Context, arg SaveGuildWarHistoryParams) error {
	_, err := q.exec(ctx, q.saveGuildWarHistoryStmt, SaveGuildWarHistory,
		arg.GuildID,
		arg.War,
		arg.Area,
		arg.GroupID,
		arg.UID,
		arg.Duration,
		arg.Record,
	)
	return err
}

const SaveMigration = `
INSERT INTO migrations(name,completed) VALUES (?1,1)
`

func (q *Queries) SaveMigration(ctx context.Context, name string) error {
	_, err := q.exec(ctx, q.saveMigrationStmt, SaveMigration, name)
	return err
}

const SaveRankEntry = `
INSERT INTO rank_entries(uid,mode,rank_id,score,achieved) VALUES (?,?,?,?,?) ON CONFLICT(uid,mode,rank_id) DO UPDATE SET score=excluded.score,achieved=excluded.achieved WHERE excluded.score>rank_entries.score
`

type SaveRankEntryParams struct {
	UID      model.UserID
	Mode     model.RankMode
	RankID   int64
	Score    int64
	Achieved int64
}

func (q *Queries) SaveRankEntry(ctx context.Context, arg SaveRankEntryParams) error {
	_, err := q.exec(ctx, q.saveRankEntryStmt, SaveRankEntry,
		arg.UID,
		arg.Mode,
		arg.RankID,
		arg.Score,
		arg.Achieved,
	)
	return err
}

const SaveRecoveredNetwork = `
UPDATE accounts SET profile=json_set(profile,'$.network',json(CAST(?1 AS TEXT))) WHERE uid=?2
`

type SaveRecoveredNetworkParams struct {
	Network string
	UID     model.UserID
}

func (q *Queries) SaveRecoveredNetwork(ctx context.Context, arg SaveRecoveredNetworkParams) error {
	_, err := q.exec(ctx, q.saveRecoveredNetworkStmt, SaveRecoveredNetwork, arg.Network, arg.UID)
	return err
}

const SaveWorldMessage = `
INSERT INTO world_messages(channel,sender,kind,content,created) VALUES (?,?,?,?,?) RETURNING id
`

type SaveWorldMessageParams struct {
	Channel string
	Sender  model.UserID
	Kind    int64
	Content string
	Created int64
}

func (q *Queries) SaveWorldMessage(ctx context.Context, arg SaveWorldMessageParams) (int64, error) {
	row := q.queryRow(ctx, q.saveWorldMessageStmt, SaveWorldMessage,
		arg.Channel,
		arg.Sender,
		arg.Kind,
		arg.Content,
		arg.Created,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const SentGiftCount = `
SELECT count(*) FROM friend_gifts WHERE day=?1 AND sender=?2
`

type SentGiftCountParams struct {
	Day    int64
	Sender model.UserID
}

func (q *Queries) SentGiftCount(ctx context.Context, arg SentGiftCountParams) (int64, error) {
	row := q.queryRow(ctx, q.sentGiftCountStmt, SentGiftCount, arg.Day, arg.Sender)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const SetFriendNickname = `
UPDATE friends SET nickname=?1 WHERE owner=?2 AND peer=?3
`

type SetFriendNicknameParams struct {
	Nickname string
	Owner    model.UserID
	Peer     model.UserID
}

func (q *Queries) SetFriendNickname(ctx context.Context, arg SetFriendNicknameParams) (int64, error) {
	result, err := q.exec(ctx, q.setFriendNicknameStmt, SetFriendNickname, arg.Nickname, arg.Owner, arg.Peer)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const SetGuildBoss = `
UPDATE guilds SET boss=?1 WHERE id=?2
`

type SetGuildBossParams struct {
	Boss model.UserID
	ID   model.GuildID
}

func (q *Queries) SetGuildBoss(ctx context.Context, arg SetGuildBossParams) error {
	_, err := q.exec(ctx, q.setGuildBossStmt, SetGuildBoss, arg.Boss, arg.ID)
	return err
}

const SetGuildLeft = `
INSERT INTO guild_player_state(uid,last_left) VALUES (?1,?2) ON CONFLICT(uid) DO UPDATE SET last_left=excluded.last_left
`

type SetGuildLeftParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetGuildLeft(ctx context.Context, arg SetGuildLeftParams) error {
	_, err := q.exec(ctx, q.setGuildLeftStmt, SetGuildLeft, arg.UID, arg.Time)
	return err
}

const SetGuildProgress = `
UPDATE guilds SET level=?,experience=? WHERE id=?
`

type SetGuildProgressParams struct {
	Level      int64
	Experience int64
	ID         model.GuildID
}

func (q *Queries) SetGuildProgress(ctx context.Context, arg SetGuildProgressParams) error {
	_, err := q.exec(ctx, q.setGuildProgressStmt, SetGuildProgress, arg.Level, arg.Experience, arg.ID)
	return err
}

const SetGuildRole = `
UPDATE guild_members SET role=?1 WHERE uid=?2
`

type SetGuildRoleParams struct {
	Role model.GuildRole
	UID  model.UserID
}

func (q *Queries) SetGuildRole(ctx context.Context, arg SetGuildRoleParams) error {
	_, err := q.exec(ctx, q.setGuildRoleStmt, SetGuildRole, arg.Role, arg.UID)
	return err
}

const SetGuildSearch = `
INSERT INTO guild_player_state(uid,last_search) VALUES (?1,?2) ON CONFLICT(uid) DO UPDATE SET last_search=excluded.last_search
`

type SetGuildSearchParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetGuildSearch(ctx context.Context, arg SetGuildSearchParams) error {
	_, err := q.exec(ctx, q.setGuildSearchStmt, SetGuildSearch, arg.UID, arg.Time)
	return err
}

const SetPresenceOffline = `
INSERT INTO presence(uid,online,offline) VALUES (?1,0,?2) ON CONFLICT(uid) DO UPDATE SET offline=excluded.offline
`

type SetPresenceOfflineParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetPresenceOffline(ctx context.Context, arg SetPresenceOfflineParams) error {
	_, err := q.exec(ctx, q.setPresenceOfflineStmt, SetPresenceOffline, arg.UID, arg.Time)
	return err
}

const SetPresenceOnline = `
INSERT INTO presence(uid,online,offline) VALUES (?1,?2,0) ON CONFLICT(uid) DO UPDATE SET online=excluded.online
`

type SetPresenceOnlineParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetPresenceOnline(ctx context.Context, arg SetPresenceOnlineParams) error {
	_, err := q.exec(ctx, q.setPresenceOnlineStmt, SetPresenceOnline, arg.UID, arg.Time)
	return err
}

const SetRecommendTime = `
INSERT INTO social_times(uid,recommend) VALUES (?1,?2) ON CONFLICT(uid) DO UPDATE SET recommend=excluded.recommend
`

type SetRecommendTimeParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetRecommendTime(ctx context.Context, arg SetRecommendTimeParams) error {
	_, err := q.exec(ctx, q.setRecommendTimeStmt, SetRecommendTime, arg.UID, arg.Time)
	return err
}

const SetSearchTime = `
INSERT INTO social_times(uid,search) VALUES (?1,?2) ON CONFLICT(uid) DO UPDATE SET search=excluded.search
`

type SetSearchTimeParams struct {
	UID  model.UserID
	Time int64
}

func (q *Queries) SetSearchTime(ctx context.Context, arg SetSearchTimeParams) error {
	_, err := q.exec(ctx, q.setSearchTimeStmt, SetSearchTime, arg.UID, arg.Time)
	return err
}

const SetSupportRefresh = `
INSERT INTO support_refresh(uid,refreshed) VALUES (?,?) ON CONFLICT(uid) DO UPDATE SET refreshed=excluded.refreshed
`

type SetSupportRefreshParams struct {
	UID       model.UserID
	Refreshed int64
}

func (q *Queries) SetSupportRefresh(ctx context.Context, arg SetSupportRefreshParams) error {
	_, err := q.exec(ctx, q.setSupportRefreshStmt, SetSupportRefresh, arg.UID, arg.Refreshed)
	return err
}

const SetWorldMembership = `
INSERT INTO world_membership(uid,channel,joined) VALUES (?,?,?) ON CONFLICT(uid) DO UPDATE SET channel=excluded.channel,joined=excluded.joined
`

type SetWorldMembershipParams struct {
	UID     model.UserID
	Channel string
	Joined  int64
}

func (q *Queries) SetWorldMembership(ctx context.Context, arg SetWorldMembershipParams) error {
	_, err := q.exec(ctx, q.setWorldMembershipStmt, SetWorldMembership, arg.UID, arg.Channel, arg.Joined)
	return err
}

const SupportOwnerUsageCount = `
SELECT CAST(coalesce(sum(uses),0) AS INTEGER) AS count FROM support_usage WHERE owner=? AND borrower=? AND day=?
`

type SupportOwnerUsageCountParams struct {
	Owner    model.UserID
	Borrower model.UserID
	Day      int64
}

func (q *Queries) SupportOwnerUsageCount(ctx context.Context, arg SupportOwnerUsageCountParams) (int64, error) {
	row := q.queryRow(ctx, q.supportOwnerUsageCountStmt, SupportOwnerUsageCount, arg.Owner, arg.Borrower, arg.Day)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const SupportUsageCount = `
SELECT CAST(coalesce(sum(uses),0) AS INTEGER) AS count FROM support_usage WHERE borrower=? AND day=?
`

type SupportUsageCountParams struct {
	Borrower model.UserID
	Day      int64
}

func (q *Queries) SupportUsageCount(ctx context.Context, arg SupportUsageCountParams) (int64, error) {
	row := q.queryRow(ctx, q.supportUsageCountStmt, SupportUsageCount, arg.Borrower, arg.Day)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const UpdateGuildAnnounce = `
UPDATE guilds SET announcement=?1 WHERE id=?2
`

type UpdateGuildAnnounceParams struct {
	Announcement string
	ID           model.GuildID
}

func (q *Queries) UpdateGuildAnnounce(ctx context.Context, arg UpdateGuildAnnounceParams) error {
	_, err := q.exec(ctx, q.updateGuildAnnounceStmt, UpdateGuildAnnounce, arg.Announcement, arg.ID)
	return err
}

const UpdateGuildSettings = `
UPDATE guilds SET name=?1,search_name=?2,signature=?3,announcement=?4,approval=?5,minimum_level=?6,head=?7,frame=?8,name_updated=?9 WHERE id=?10
`

type UpdateGuildSettingsParams struct {
	Name         string
	SearchName   string
	Signature    string
	Announcement string
	Approval     int64
	MinimumLevel int64
	Head         int64
	Frame        int64
	NameUpdated  int64
	ID           model.GuildID
}

func (q *Queries) UpdateGuildSettings(ctx context.Context, arg UpdateGuildSettingsParams) error {
	_, err := q.exec(ctx, q.updateGuildSettingsStmt, UpdateGuildSettings,
		arg.Name,
		arg.SearchName,
		arg.Signature,
		arg.Announcement,
		arg.Approval,
		arg.MinimumLevel,
		arg.Head,
		arg.Frame,
		arg.NameUpdated,
		arg.ID,
	)
	return err
}

const UpsertAccount = `
INSERT INTO accounts(uid,login,display_name,search_name,level,last_login,profile) VALUES (?1,?2,?3,?4,?5,?6,?7) ON CONFLICT(uid) DO UPDATE SET display_name=excluded.display_name,search_name=excluded.search_name,level=excluded.level,last_login=excluded.last_login,profile=excluded.profile WHERE accounts.profile!=excluded.profile
`

type UpsertAccountParams struct {
	UID         model.UserID
	Login       string
	DisplayName string
	SearchName  string
	Level       int64
	LastLogin   int64
	Profile     []byte
}

func (q *Queries) UpsertAccount(ctx context.Context, arg UpsertAccountParams) error {
	_, err := q.exec(ctx, q.upsertAccountStmt, UpsertAccount,
		arg.UID,
		arg.Login,
		arg.DisplayName,
		arg.SearchName,
		arg.Level,
		arg.LastLogin,
		arg.Profile,
	)
	return err
}

const UpsertRecord = `
INSERT INTO account_records(uid,category,id,payload) VALUES (?1,?2,?3,?4) ON CONFLICT(uid,category,id) DO UPDATE SET payload=excluded.payload
`

type UpsertRecordParams struct {
	UID      model.UserID
	Category model.RecordCategory
	ID       string
	Payload  []byte
}

func (q *Queries) UpsertRecord(ctx context.Context, arg UpsertRecordParams) error {
	_, err := q.exec(ctx, q.upsertRecordStmt, UpsertRecord,
		arg.UID,
		arg.Category,
		arg.ID,
		arg.Payload,
	)
	return err
}
