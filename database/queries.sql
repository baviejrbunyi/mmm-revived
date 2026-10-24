-- name: GetAccount :one
SELECT uid,profile FROM accounts WHERE login=:login;

-- name: GetCredential :one
SELECT uid,version,memory_kib,iterations,parallelism,salt,digest,created FROM account_credentials WHERE uid=:uid;

-- name: ClaimCredential :execrows
INSERT INTO account_credentials(uid,version,memory_kib,iterations,parallelism,salt,digest,created) VALUES (:uid,:version,:memory_kib,:iterations,:parallelism,:salt,:digest,:created) ON CONFLICT(uid) DO NOTHING;

-- name: GetMigration :one
SELECT completed FROM migrations WHERE name=:name;

-- name: AccountExists :one
SELECT count(*) FROM accounts WHERE uid=:uid;

-- name: SaveMigration :exec
INSERT INTO migrations(name,completed) VALUES (:name,1);

-- name: UpsertAccount :exec
INSERT INTO accounts(uid,login,display_name,search_name,level,last_login,profile) VALUES (:uid,:login,:display_name,:search_name,:level,:last_login,:profile) ON CONFLICT(uid) DO UPDATE SET display_name=excluded.display_name,search_name=excluded.search_name,level=excluded.level,last_login=excluded.last_login,profile=excluded.profile WHERE accounts.profile!=excluded.profile;

-- name: UpsertRecord :exec
INSERT INTO account_records(uid,category,id,payload) VALUES (:uid,:category,:id,:payload) ON CONFLICT(uid,category,id) DO UPDATE SET payload=excluded.payload;

-- name: DeleteRecord :exec
DELETE FROM account_records WHERE uid=:uid AND category=:category AND id=:id;

-- name: ListRecords :many
SELECT category,id,payload FROM account_records WHERE uid=:uid;

-- name: SetPresenceOnline :exec
INSERT INTO presence(uid,online,offline) VALUES (:uid,:time,0) ON CONFLICT(uid) DO UPDATE SET online=excluded.online;

-- name: SetPresenceOffline :exec
INSERT INTO presence(uid,online,offline) VALUES (:uid,0,:time) ON CONFLICT(uid) DO UPDATE SET offline=excluded.offline;

-- name: ListFriends :many
SELECT peer,nickname,added FROM friends WHERE owner=:owner;

-- name: ListApplications :many
SELECT sender,source,created FROM friend_requests WHERE receiver=:receiver;

-- name: ListBlocks :many
SELECT peer FROM blocks WHERE owner=:owner ORDER BY peer;

-- name: ListReceivedGifts :many
SELECT sender,claimed FROM friend_gifts WHERE receiver=:receiver AND day=:day;

-- name: ListSentGifts :many
SELECT receiver FROM friend_gifts WHERE sender=:sender AND day=:day;

-- name: GetSocialTimes :one
SELECT search,recommend FROM social_times WHERE uid=:uid;

-- name: FriendshipExists :one
SELECT count(*) FROM friends WHERE owner=:owner AND peer=:peer;

-- name: BlockEither :one
SELECT count(*) FROM blocks WHERE (owner=:owner AND peer=:peer) OR (owner=:peer AND peer=:owner);

-- name: FriendRequestExists :one
SELECT count(*) FROM friend_requests WHERE receiver=:receiver AND sender=:sender;

-- name: FriendCount :one
SELECT count(*) FROM friends WHERE owner=:owner;

-- name: RequestCount :one
SELECT count(*) FROM friend_requests WHERE receiver=:receiver;

-- name: InsertFriendRequest :exec
INSERT INTO friend_requests(receiver,sender,source,created) VALUES (:receiver,:sender,:source,:created) ON CONFLICT(receiver,sender) DO NOTHING;

-- name: InsertFriendPair :exec
INSERT INTO friends(owner,peer,added) VALUES (:owner,:peer,:added),(:peer,:owner,:added) ON CONFLICT(owner,peer) DO NOTHING;

-- name: DeleteFriendRequestPair :exec
DELETE FROM friend_requests WHERE (receiver=:receiver AND sender=:sender) OR (receiver=:sender AND sender=:receiver);

-- name: DeleteFriendRequest :exec
DELETE FROM friend_requests WHERE receiver=:receiver AND sender=:sender;

-- name: DeleteAllFriendRequests :exec
DELETE FROM friend_requests WHERE receiver=:receiver;

-- name: DeleteFriendPair :exec
DELETE FROM friends WHERE (owner=:owner AND peer=:peer) OR (owner=:peer AND peer=:owner);

-- name: InsertBlock :exec
INSERT INTO blocks(owner,peer,created) VALUES (:owner,:peer,:created) ON CONFLICT(owner,peer) DO NOTHING;

-- name: DeleteBlock :exec
DELETE FROM blocks WHERE owner=:owner AND peer=:peer;

-- name: SetFriendNickname :execrows
UPDATE friends SET nickname=:nickname WHERE owner=:owner AND peer=:peer;

-- name: FindPlayerByID :many
SELECT a.uid FROM accounts a WHERE a.uid=:target AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=a.uid AND b.peer=:self) LIMIT 1;

-- name: FindPlayersByName :many
SELECT a.uid FROM accounts a WHERE NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=a.uid AND b.peer=:self) AND a.search_name GLOB :pattern ORDER BY a.search_name,a.uid LIMIT 20;

-- name: RecommendPlayers :many
SELECT a.uid FROM accounts a WHERE a.uid!=:self AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.owner=:self AND b.peer=a.uid) OR (b.owner=a.uid AND b.peer=:self)) AND NOT EXISTS(SELECT 1 FROM friends f WHERE f.owner=:self AND f.peer=a.uid) AND NOT EXISTS(SELECT 1 FROM friend_requests q WHERE (q.receiver=a.uid AND q.sender=:self) OR (q.receiver=:self AND q.sender=a.uid)) ORDER BY a.last_login DESC,a.uid LIMIT 20;

-- name: SetSearchTime :exec
INSERT INTO social_times(uid,search) VALUES (:uid,:time) ON CONFLICT(uid) DO UPDATE SET search=excluded.search;

-- name: SetRecommendTime :exec
INSERT INTO social_times(uid,recommend) VALUES (:uid,:time) ON CONFLICT(uid) DO UPDATE SET recommend=excluded.recommend;

-- name: SentGiftCount :one
SELECT count(*) FROM friend_gifts WHERE day=:day AND sender=:sender;

-- name: GiftExists :one
SELECT count(*) FROM friend_gifts WHERE day=:day AND sender=:sender AND receiver=:receiver;

-- name: InsertGift :exec
INSERT INTO friend_gifts(day,sender,receiver) VALUES (:day,:sender,:receiver);

-- name: ReceivedGiftCount :one
SELECT count(*) FROM friend_gifts WHERE day=:day AND receiver=:receiver AND claimed=1;

-- name: ClaimableGift :one
SELECT count(*) FROM friend_gifts g WHERE g.day=:day AND g.receiver=:receiver AND g.sender=:sender AND g.claimed=0 AND EXISTS(SELECT 1 FROM friends f WHERE f.owner=g.receiver AND f.peer=g.sender);

-- name: MarkGiftClaimed :exec
UPDATE friend_gifts SET claimed=1 WHERE day=:day AND receiver=:receiver AND sender=:sender;

-- name: InsertPrivateMessage :one
INSERT INTO chat_messages(room,channel,sender,receiver,kind,content,created) VALUES (:room,1,:sender,:receiver,:kind,:content,:created) RETURNING id;

-- name: ListPendingMessages :many
SELECT id,sender,receiver,kind,content,created FROM chat_messages c WHERE receiver=:receiver AND channel=1 AND acknowledged=0 AND EXISTS(SELECT 1 FROM friends f WHERE f.owner=c.receiver AND f.peer=c.sender) ORDER BY id DESC LIMIT :max_rows;

-- name: AcknowledgeAllMessages :exec
UPDATE chat_messages SET acknowledged=1 WHERE receiver=:receiver AND channel=1 AND acknowledged=0;

-- name: AcknowledgePeerMessages :exec
UPDATE chat_messages SET acknowledged=1 WHERE receiver=:receiver AND sender=:sender AND channel=1 AND acknowledged=0;

-- name: AcknowledgePairMessages :exec
UPDATE chat_messages SET acknowledged=1 WHERE channel=1 AND ((sender=:sender AND receiver=:receiver) OR (sender=:receiver AND receiver=:sender));

-- name: GetPublicProfiles :many
SELECT a.uid,a.profile,COALESCE(p.online,0) AS online,COALESCE(p.offline,0) AS offline,(SELECT count(*) FROM account_records h WHERE h.uid=a.uid AND h.category='hero') AS hero_count,COALESCE(m.guild_id,0) AS guild_id,COALESCE(m.role,'') AS guild_role
FROM accounts a LEFT JOIN presence p ON p.uid=a.uid LEFT JOIN guild_members m ON m.uid=a.uid
WHERE a.uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(:uids AS TEXT)));

-- name: GetPublicHeroes :many
SELECT uid,id,payload FROM account_records WHERE category='hero' AND uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(:uids AS TEXT)));

-- name: CreateGuild :one
INSERT INTO guilds(name,search_name,signature,announcement,approval,minimum_level,head,frame,boss,created,name_updated) VALUES (:name,:search_name,:signature,:announcement,:approval,:minimum_level,:head,:frame,:boss,:created,:created) RETURNING id;

-- name: GetGuild :one
SELECT * FROM guilds WHERE id=:id;

-- name: GetGuildMembership :one
SELECT * FROM guild_members WHERE uid=:uid;

-- name: InsertGuildMember :exec
INSERT INTO guild_members(uid,guild_id,role,joined) VALUES (:uid,:guild_id,:role,:joined);

-- name: DeleteGuildMember :exec
DELETE FROM guild_members WHERE uid=:uid;

-- name: DeleteGuild :exec
DELETE FROM guilds WHERE id=:id;

-- name: SetGuildRole :exec
UPDATE guild_members SET role=:role WHERE uid=:uid;

-- name: SetGuildBoss :exec
UPDATE guilds SET boss=:boss WHERE id=:id;

-- name: GuildAdminCount :one
SELECT count(*) FROM guild_members WHERE guild_id=:guild_id AND role='manager';

-- name: ListGuildMembers :many
SELECT m.uid,m.role,m.joined,m.score,m.history_score,COALESCE(p.online,0) AS online,COALESCE(p.offline,0) AS offline FROM guild_members m LEFT JOIN presence p ON m.uid=p.uid WHERE m.guild_id=:guild_id ORDER BY m.uid;

-- name: GetGuilds :many
SELECT * FROM guilds WHERE id IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(:ids AS TEXT)));

-- name: FindGuildsByName :many
SELECT id FROM guilds WHERE search_name GLOB :pattern ORDER BY search_name,id LIMIT 20;

-- name: RecommendGuilds :many
SELECT id FROM guilds WHERE minimum_level<=:level AND member_count<:capacity ORDER BY member_count DESC,id LIMIT 20;

-- name: UpdateGuildSettings :exec
UPDATE guilds SET name=:name,search_name=:search_name,signature=:signature,announcement=:announcement,approval=:approval,minimum_level=:minimum_level,head=:head,frame=:frame,name_updated=:name_updated WHERE id=:id;

-- name: UpdateGuildAnnounce :exec
UPDATE guilds SET announcement=:announcement WHERE id=:id;

-- name: GetGuildPlayerState :one
SELECT * FROM guild_player_state WHERE uid=:uid;

-- name: SetGuildLeft :exec
INSERT INTO guild_player_state(uid,last_left) VALUES (:uid,:time) ON CONFLICT(uid) DO UPDATE SET last_left=excluded.last_left;

-- name: SetGuildSearch :exec
INSERT INTO guild_player_state(uid,last_search) VALUES (:uid,:time) ON CONFLICT(uid) DO UPDATE SET last_search=excluded.last_search;

-- name: ListPlayerGuildApplications :many
SELECT guild_id,created FROM guild_applications WHERE applicant=:applicant AND created>:after;

-- name: ListGuildApplications :many
SELECT applicant,created FROM guild_applications WHERE guild_id=:guild_id AND created>:after;

-- name: GetGuildApplication :one
SELECT created FROM guild_applications WHERE guild_id=:guild_id AND applicant=:applicant;

-- name: InsertGuildApplication :exec
INSERT INTO guild_applications(guild_id,applicant,created) VALUES (:guild_id,:applicant,:created) ON CONFLICT(guild_id,applicant) DO UPDATE SET created=excluded.created;

-- name: DeleteGuildApplication :exec
DELETE FROM guild_applications WHERE guild_id=:guild_id AND applicant=:applicant;

-- name: DeletePlayerGuildApplications :exec
DELETE FROM guild_applications WHERE applicant=:applicant;

-- name: ListPlayerGuildInvites :many
SELECT guild_id,created FROM guild_invites WHERE player=:player AND created>:after;

-- name: ListGuildInvites :many
SELECT player,created FROM guild_invites WHERE guild_id=:guild_id AND created>:after;

-- name: GetGuildInvite :one
SELECT created FROM guild_invites WHERE guild_id=:guild_id AND player=:player;

-- name: InsertGuildInvite :exec
INSERT INTO guild_invites(guild_id,player,inviter,created) VALUES (:guild_id,:player,:inviter,:created) ON CONFLICT(guild_id,player) DO UPDATE SET inviter=excluded.inviter,created=excluded.created;

-- name: DeleteGuildInvite :exec
DELETE FROM guild_invites WHERE guild_id=:guild_id AND player=:player;

-- name: DeletePlayerGuildInvites :exec
DELETE FROM guild_invites WHERE player=:player;

-- name: InsertGuildNews :one
INSERT INTO guild_news(guild_id,kind,actor,subject,value,created) VALUES (:guild_id,:kind,:actor,:subject,:value,:created) RETURNING id;

-- name: ListGuildNews :many
SELECT * FROM guild_news WHERE guild_id=:guild_id ORDER BY id DESC LIMIT 100;

-- name: InsertGuildReport :exec
INSERT INTO guild_reports(reporter,guild_id,reason,content,created) VALUES (:reporter,:guild_id,:reason,:content,:created);

-- name: InsertGuildMessage :one
INSERT INTO guild_chat(guild_id,sender,kind,content,created) VALUES (:guild_id,:sender,:kind,:content,:created) RETURNING id;

-- name: ListGuildMessages :many
SELECT id,sender,kind,content,created FROM guild_chat WHERE guild_id=:guild_id ORDER BY id DESC LIMIT :max_rows;

-- name: GuildNameExists :one
SELECT count(*) FROM guilds WHERE name=:name COLLATE NOCASE;

-- name: PlayerLevel :one
SELECT level FROM accounts WHERE uid=:uid;

-- name: SaveCommunityMessage :exec
INSERT INTO community_messages(scope,subject,uid,content,updated) VALUES (:scope,:subject,:uid,:content,:updated) ON CONFLICT(scope,subject,uid) DO UPDATE SET content=excluded.content,updated=excluded.updated;

-- name: ListCommunityMessages :many
SELECT uid,content,updated FROM community_messages WHERE scope=:scope AND subject=:subject ORDER BY updated DESC,uid LIMIT :max_rows;

-- name: DeliverMail :one
INSERT INTO mailbox(owner,title,content,sender,template,created,expires,reward_id,items,has_reward) VALUES (:owner,:title,:content,:sender,:template,:created,:expires,:reward_id,:items,:has_reward) RETURNING id;

-- name: MailboxCount :one
SELECT count(*) FROM mailbox WHERE owner=:owner;

-- name: ListMailbox :many
SELECT * FROM mailbox WHERE owner=:owner AND (expires=0 OR expires>:now) ORDER BY id DESC LIMIT :max_rows;

-- name: GetMail :one
SELECT * FROM mailbox WHERE owner=:owner AND id=:id;

-- name: OpenMail :exec
UPDATE mailbox SET status=2 WHERE owner=:owner AND id=:id AND status=1 AND (expires=0 OR expires>:now);

-- name: ClaimMail :execrows
UPDATE mailbox SET status=3 WHERE owner=:owner AND id=:id AND status<3 AND (expires=0 OR expires>:now);

-- name: DeleteClaimedMail :execrows
DELETE FROM mailbox WHERE owner=:owner AND id=:id AND (status=3 OR has_reward=0 OR (expires>0 AND expires<=:now));

-- name: DeleteAllClaimedMail :exec
DELETE FROM mailbox WHERE owner=:owner AND (status=3 OR (expires>0 AND expires<=:now));

-- name: ListAccountUIDs :many
SELECT uid FROM accounts ORDER BY uid;

-- name: NextAccountUID :one
SELECT max(COALESCE(max(uid),0),100000)+1 FROM accounts;

-- name: MigrateAccountUID :exec
UPDATE accounts SET uid=:new_uid,profile=json_set(profile,'$.uid',:new_uid) WHERE uid=:old_uid;

-- name: ResolveLegacyUID :one
SELECT new_uid FROM uid_history WHERE old_uid=:old_uid;


-- name: ListSupportUsage :many
SELECT borrower,hero,uses,claimed,day FROM support_usage WHERE owner=? ORDER BY day DESC,borrower,hero;

-- name: SupportUsageCount :one
SELECT CAST(coalesce(sum(uses),0) AS INTEGER) AS count FROM support_usage WHERE borrower=? AND day=?;

-- name: AddSupportUsage :exec
INSERT INTO support_usage(owner,borrower,hero,day) VALUES (?,?,?,?) ON CONFLICT(owner,borrower,hero,day) DO UPDATE SET uses=uses+1;

-- name: ClaimSupportUsage :execrows
UPDATE support_usage SET claimed=uses WHERE owner=? AND claimed<uses;

-- name: GetSupportRefresh :one
SELECT refreshed FROM support_refresh WHERE uid=?;

-- name: SetSupportRefresh :exec
INSERT INTO support_refresh(uid,refreshed) VALUES (?,?) ON CONFLICT(uid) DO UPDATE SET refreshed=excluded.refreshed;

-- name: ListSupportCandidates :many
SELECT a.uid FROM accounts a WHERE a.uid<>sqlc.arg(uid) AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.owner=sqlc.arg(uid) AND b.peer=a.uid) ORDER BY EXISTS(SELECT 1 FROM friends f WHERE f.owner=?1 AND f.peer=a.uid) DESC,a.last_login DESC,a.uid LIMIT 30;

-- name: ListAsyncCandidates :many
SELECT a.uid FROM accounts a WHERE a.uid<>sqlc.arg(uid) AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.owner=sqlc.arg(uid) AND b.peer=a.uid) ORDER BY a.last_login DESC,a.uid LIMIT 30;

-- name: SetWorldMembership :exec
INSERT INTO world_membership(uid,channel,joined) VALUES (?,?,?) ON CONFLICT(uid) DO UPDATE SET channel=excluded.channel,joined=excluded.joined;

-- name: GetWorldMembership :one
SELECT channel FROM world_membership WHERE uid=?;

-- name: ListWorldMembers :many
SELECT uid FROM world_membership WHERE channel=? ORDER BY uid;

-- name: CountWorldMembers :one
SELECT count(*) FROM world_membership WHERE channel=?;

-- name: SaveWorldMessage :one
INSERT INTO world_messages(channel,sender,kind,content,created) VALUES (?,?,?,?,?) RETURNING id;

-- name: ListWorldMessages :many
SELECT id,sender,kind,content,created FROM world_messages WHERE channel=? ORDER BY id DESC LIMIT 100;

-- name: PruneWorldMessages :exec
DELETE FROM world_messages WHERE world_messages.channel=sqlc.arg(channel) AND id < coalesce((SELECT w.id FROM world_messages w WHERE w.channel=sqlc.arg(channel) ORDER BY id DESC LIMIT 1 OFFSET 199),0);

-- name: GetGuildWar :one
SELECT state FROM guild_wars WHERE guild_id=? AND war=?;

-- name: SaveGuildWar :exec
INSERT INTO guild_wars(guild_id,war,state) VALUES (?,?,?) ON CONFLICT(guild_id,war) DO UPDATE SET state=excluded.state;

-- name: SaveGuildWarHistory :exec
INSERT INTO guild_war_history(guild_id,war,area,group_id,uid,duration,record) VALUES (?,?,?,?,?,?,?);

-- name: ListGuildWarHistory :many
SELECT record FROM guild_war_history WHERE guild_id=? AND war=? AND area=? AND group_id=? ORDER BY duration,id LIMIT 50;

-- name: AddGuildWarProgress :exec
INSERT INTO guild_war_progress(guild_id,war,uid,area,score,assist,score_time,assist_time) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(guild_id,war,uid,area) DO UPDATE SET score=score+excluded.score, assist=assist+excluded.assist, score_time=max(score_time,excluded.score_time), assist_time=max(assist_time,excluded.assist_time);

-- name: ListGuildWarProgress :many
SELECT uid, area, score, assist, score_time, assist_time FROM guild_war_progress WHERE guild_id=? AND war=?;

-- name: SaveAsyncRecord :exec
INSERT INTO async_records(attacker,defender,opponent,friend,created,record) VALUES (?,?,?,?,?,?);

-- name: ListAsyncRecords :many
SELECT record FROM async_records WHERE (attacker=sqlc.arg(uid) OR defender=sqlc.arg(uid)) AND friend=sqlc.arg(friend) ORDER BY id DESC LIMIT 50;

-- name: SupportOwnerUsageCount :one
SELECT CAST(coalesce(sum(uses),0) AS INTEGER) AS count FROM support_usage WHERE owner=? AND borrower=? AND day=?;

-- name: GetPublicEquipment :many
SELECT uid,id,category,payload FROM account_records WHERE uid IN (SELECT CAST(value AS INTEGER) FROM json_each(CAST(:uids AS TEXT))) AND category IN ('artifact','highlight') AND coalesce(json_extract(payload,'$.Install'),0)>0;

-- name: ReadClockOffset :one
SELECT offset_seconds FROM server_clock WHERE singleton=1;

-- name: SaveClockOffset :exec
INSERT INTO server_clock(singleton,offset_seconds) VALUES (1,?) ON CONFLICT(singleton) DO UPDATE SET offset_seconds=excluded.offset_seconds;

-- name: ListPendingNetworkAccounts :many
SELECT uid, CAST(json_extract(profile,'$.network') AS BLOB) AS network FROM accounts WHERE json_type(profile,'$.network.pending')='object';

-- name: SaveRecoveredNetwork :exec
UPDATE accounts SET profile=json_set(profile,'$.network',json(CAST(sqlc.arg(network) AS TEXT))) WHERE uid=sqlc.arg(uid);

-- name: SaveRankEntry :exec
INSERT INTO rank_entries(uid,mode,rank_id,score,achieved) VALUES (?,?,?,?,?) ON CONFLICT(uid,mode,rank_id) DO UPDATE SET score=excluded.score,achieved=excluded.achieved WHERE excluded.score>rank_entries.score;

-- name: GetRankEntry :one
SELECT score,achieved FROM rank_entries WHERE uid=? AND mode=? AND rank_id=?;

-- name: ListFriendRanks :many
SELECT e.uid,e.score,e.achieved FROM rank_entries e JOIN friends f ON f.peer=e.uid WHERE f.owner=? AND e.mode=? AND e.rank_id=? ORDER BY e.score DESC,e.achieved,e.uid;

-- name: ListGuildRanks :many
SELECT e.uid,e.score,e.achieved FROM rank_entries e JOIN guild_members m ON m.uid=e.uid WHERE m.guild_id=(SELECT own.guild_id FROM guild_members own WHERE own.uid=?) AND e.mode=? AND e.rank_id=? ORDER BY e.score DESC,e.achieved,e.uid;

-- name: RankBoardSize :one
SELECT count(*) FROM rank_entries WHERE mode=? AND rank_id=?;

-- name: RankHigherCount :one
SELECT count(*) FROM rank_entries WHERE mode=sqlc.arg(mode) AND rank_id=sqlc.arg(rank_id) AND (score>sqlc.arg(score) OR (score=sqlc.arg(score) AND achieved<sqlc.arg(achieved)) OR (score=sqlc.arg(score) AND achieved=sqlc.arg(achieved) AND uid<sqlc.arg(uid)));

-- name: ListPlayerRanks :many
SELECT mode,rank_id,score,achieved FROM rank_entries WHERE uid=? ORDER BY mode,rank_id;

-- name: GetInviter :one
SELECT inviter,bound FROM invite_links WHERE invitee=?;

-- name: ListInvitees :many
SELECT invitee FROM invite_links WHERE inviter=? ORDER BY invitee;

-- name: InsertInvite :exec
INSERT INTO invite_links(invitee,inviter,bound) VALUES (?,?,?);


-- name: GetAccountLevel :one
SELECT level FROM accounts WHERE uid=?;

-- name: GetInviteProgress :many
SELECT l.invitee,l.bound,a.level,
CAST(EXISTS(SELECT 1 FROM account_records p WHERE p.uid=l.invitee AND p.category='passed' AND p.id='100109' AND coalesce(json_extract(p.payload,'$.stars'),0)>0) AS INTEGER) AS story,
CAST(coalesce(json_extract(t1.payload,'$.status'),0) AS INTEGER) AS purchase1,
CAST(coalesce(json_extract(t2.payload,'$.status'),0) AS INTEGER) AS purchase2,
CAST(coalesce(json_extract(t3.payload,'$.status'),0) AS INTEGER) AS purchase3
FROM invite_links l JOIN accounts a ON a.uid=l.invitee
LEFT JOIN account_records t1 ON t1.uid=l.invitee AND t1.category='task' AND t1.id='3711'
LEFT JOIN account_records t2 ON t2.uid=l.invitee AND t2.category='task' AND t2.id='3712'
LEFT JOIN account_records t3 ON t3.uid=l.invitee AND t3.category='task' AND t3.id='3713'
WHERE l.inviter=? ORDER BY l.invitee LIMIT 5;

-- name: AddGuildContribution :exec
INSERT INTO guild_contributions(guild_id,uid,week,score) VALUES (?,?,?,?) ON CONFLICT(guild_id,week,uid) DO UPDATE SET score=score+excluded.score;

-- name: AddGuildHistoryScore :exec
UPDATE guild_members SET history_score=history_score+sqlc.arg(score) WHERE uid=sqlc.arg(uid);

-- name: DeleteGuildContributions :exec
DELETE FROM guild_contributions WHERE guild_id=? AND week<?;

-- name: ListGuildContributions :many
SELECT uid,week,score FROM guild_contributions WHERE guild_id=sqlc.arg(guild_id) AND week>=sqlc.arg(from_week) AND week<sqlc.arg(to_week) ORDER BY week,uid;

-- name: SetGuildProgress :exec
UPDATE guilds SET level=?,experience=? WHERE id=?;
