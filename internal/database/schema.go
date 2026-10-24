package database

const Schema = `CREATE TABLE IF NOT EXISTS accounts (
    uid INTEGER PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL COLLATE NOCASE,
    search_name TEXT NOT NULL,
    level INTEGER NOT NULL,
    last_login INTEGER NOT NULL,
    profile BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS accounts_search ON accounts(search_name,uid);

CREATE INDEX IF NOT EXISTS accounts_name ON accounts(display_name COLLATE NOCASE,uid);

CREATE INDEX IF NOT EXISTS accounts_recent ON accounts(last_login DESC,uid);

CREATE TABLE IF NOT EXISTS account_credentials (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid) ON DELETE CASCADE ON UPDATE CASCADE,
    version INTEGER NOT NULL,
    memory_kib INTEGER NOT NULL,
    iterations INTEGER NOT NULL,
    parallelism INTEGER NOT NULL,
    salt BLOB NOT NULL CHECK(length(salt)=16),
    digest BLOB NOT NULL CHECK(length(digest)=32),
    created INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS account_records (
    uid INTEGER NOT NULL REFERENCES accounts(uid) ON DELETE CASCADE,
    category TEXT NOT NULL,
    id TEXT NOT NULL,
    payload BLOB NOT NULL,
    PRIMARY KEY(uid,category,id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS migrations (
    name TEXT PRIMARY KEY,
    completed INTEGER NOT NULL
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS friends (
    owner INTEGER NOT NULL REFERENCES accounts(uid),
    peer INTEGER NOT NULL REFERENCES accounts(uid),
    nickname TEXT NOT NULL DEFAULT '',
    added INTEGER NOT NULL,
    PRIMARY KEY(owner,peer),
    CHECK(owner!=peer)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS friend_requests (
    receiver INTEGER NOT NULL REFERENCES accounts(uid),
    sender INTEGER NOT NULL REFERENCES accounts(uid),
    source INTEGER NOT NULL,
    created INTEGER NOT NULL,
    PRIMARY KEY(receiver,sender),
    CHECK(receiver!=sender)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS friend_requests_sender ON friend_requests(sender,receiver);

CREATE TABLE IF NOT EXISTS blocks (
    owner INTEGER NOT NULL REFERENCES accounts(uid),
    peer INTEGER NOT NULL REFERENCES accounts(uid),
    created INTEGER NOT NULL,
    PRIMARY KEY(owner,peer),
    CHECK(owner!=peer)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS blocks_peer ON blocks(peer,owner);

CREATE TABLE IF NOT EXISTS friend_gifts (
    day INTEGER NOT NULL,
    sender INTEGER NOT NULL REFERENCES accounts(uid),
    receiver INTEGER NOT NULL REFERENCES accounts(uid),
    claimed INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(day,sender,receiver),
    CHECK(sender!=receiver)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS gifts_receiver ON friend_gifts(receiver,day,claimed,sender);

CREATE TABLE IF NOT EXISTS chat_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room TEXT NOT NULL,
    channel INTEGER NOT NULL,
    sender INTEGER NOT NULL REFERENCES accounts(uid),
    receiver INTEGER NOT NULL DEFAULT 0,
    kind INTEGER NOT NULL,
    content TEXT NOT NULL,
    created INTEGER NOT NULL,
    acknowledged INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS chat_room ON chat_messages(room,id DESC);

CREATE INDEX IF NOT EXISTS chat_inbox ON chat_messages(receiver,acknowledged,id);

CREATE TABLE IF NOT EXISTS presence (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    online INTEGER NOT NULL,
    offline INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS social_times (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    search INTEGER NOT NULL DEFAULT 0,
    recommend INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS guilds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    search_name TEXT NOT NULL,
    signature TEXT NOT NULL,
    announcement TEXT NOT NULL,
    approval INTEGER NOT NULL,
    minimum_level INTEGER NOT NULL,
    head INTEGER NOT NULL,
    frame INTEGER NOT NULL,
    boss INTEGER NOT NULL REFERENCES accounts(uid),
    level INTEGER NOT NULL DEFAULT 1,
    experience INTEGER NOT NULL DEFAULT 0,
    member_count INTEGER NOT NULL DEFAULT 0,
    created INTEGER NOT NULL,
    name_updated INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS guild_search ON guilds(search_name,id);
CREATE INDEX IF NOT EXISTS guild_recommended ON guilds(member_count DESC,id);
CREATE TABLE IF NOT EXISTS guild_members (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('normal','manager','boss')),
    joined INTEGER NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    history_score INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS guild_members_by_guild ON guild_members(guild_id,uid);
CREATE TRIGGER IF NOT EXISTS guild_member_added
AFTER INSERT ON guild_members
BEGIN
    UPDATE guilds SET member_count=member_count+1 WHERE id=NEW.guild_id;
END;
CREATE TRIGGER IF NOT EXISTS guild_member_removed
AFTER DELETE ON guild_members
BEGIN
    UPDATE guilds SET member_count=member_count-1 WHERE id=OLD.guild_id;
END;
CREATE TABLE IF NOT EXISTS guild_applications (
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    applicant INTEGER NOT NULL REFERENCES accounts(uid),
    created INTEGER NOT NULL,
    PRIMARY KEY(guild_id,applicant)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS guild_applicant ON guild_applications(applicant,guild_id);
CREATE TABLE IF NOT EXISTS guild_invites (
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    player INTEGER NOT NULL REFERENCES accounts(uid),
    inviter INTEGER NOT NULL REFERENCES accounts(uid),
    created INTEGER NOT NULL,
    PRIMARY KEY(guild_id,player)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS guild_invited_player ON guild_invites(player,guild_id);
CREATE TABLE IF NOT EXISTS guild_player_state (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    last_left INTEGER NOT NULL DEFAULT 0,
    last_search INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS guild_news (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    kind INTEGER NOT NULL,
    actor INTEGER NOT NULL REFERENCES accounts(uid),
    subject INTEGER NOT NULL DEFAULT 0,
    value INTEGER NOT NULL DEFAULT 0,
    created INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS guild_news_recent ON guild_news(guild_id,id DESC);
CREATE TABLE IF NOT EXISTS guild_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter INTEGER NOT NULL REFERENCES accounts(uid),
    guild_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    content TEXT NOT NULL,
    created INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS guild_chat (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    sender INTEGER NOT NULL REFERENCES accounts(uid),
    kind INTEGER NOT NULL,
    content TEXT NOT NULL,
    created INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS guild_chat_recent ON guild_chat(guild_id,id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS guild_single_boss ON guild_members(guild_id) WHERE role='boss';

CREATE TABLE IF NOT EXISTS community_messages (
    scope TEXT NOT NULL,
    subject INTEGER NOT NULL,
    uid INTEGER NOT NULL REFERENCES accounts(uid),
    content TEXT NOT NULL,
    updated INTEGER NOT NULL,
    PRIMARY KEY(scope,subject,uid)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS community_messages_recent ON community_messages(scope,subject,updated DESC,uid);

CREATE TABLE IF NOT EXISTS mailbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner INTEGER NOT NULL REFERENCES accounts(uid),
    title BLOB NOT NULL,
    content BLOB NOT NULL,
    sender BLOB NOT NULL,
    template INTEGER NOT NULL,
    created INTEGER NOT NULL,
    expires INTEGER NOT NULL,
    status INTEGER NOT NULL DEFAULT 1 CHECK(status IN (1,2,3)),
    reward_id INTEGER NOT NULL,
    items BLOB NOT NULL,
    has_reward INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS mailbox_owner ON mailbox(owner,id DESC);

CREATE TABLE IF NOT EXISTS uid_history (
    old_uid INTEGER PRIMARY KEY,
    new_uid INTEGER NOT NULL UNIQUE REFERENCES accounts(uid)
);

CREATE TRIGGER IF NOT EXISTS account_uid_changed
AFTER UPDATE OF uid ON accounts
WHEN NEW.uid != OLD.uid
BEGIN
    UPDATE account_records SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE friends SET owner=NEW.uid WHERE owner=OLD.uid;
    UPDATE friends SET peer=NEW.uid WHERE peer=OLD.uid;
    UPDATE friend_requests SET receiver=NEW.uid WHERE receiver=OLD.uid;
    UPDATE friend_requests SET sender=NEW.uid WHERE sender=OLD.uid;
    UPDATE blocks SET owner=NEW.uid WHERE owner=OLD.uid;
    UPDATE blocks SET peer=NEW.uid WHERE peer=OLD.uid;
    UPDATE friend_gifts SET sender=NEW.uid WHERE sender=OLD.uid;
    UPDATE friend_gifts SET receiver=NEW.uid WHERE receiver=OLD.uid;
    UPDATE chat_messages SET sender=NEW.uid WHERE sender=OLD.uid;
    UPDATE chat_messages SET receiver=NEW.uid WHERE receiver=OLD.uid;
    UPDATE presence SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE social_times SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE guilds SET boss=NEW.uid WHERE boss=OLD.uid;
    UPDATE guild_members SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE guild_applications SET applicant=NEW.uid WHERE applicant=OLD.uid;
    UPDATE guild_invites SET player=NEW.uid WHERE player=OLD.uid;
    UPDATE guild_invites SET inviter=NEW.uid WHERE inviter=OLD.uid;
    UPDATE guild_player_state SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE guild_news SET actor=NEW.uid WHERE actor=OLD.uid;
    UPDATE guild_news SET subject=NEW.uid WHERE subject=OLD.uid;
    UPDATE guild_reports SET reporter=NEW.uid WHERE reporter=OLD.uid;
    UPDATE guild_chat SET sender=NEW.uid WHERE sender=OLD.uid;
    UPDATE community_messages SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE mailbox SET owner=NEW.uid WHERE owner=OLD.uid;
    UPDATE account_records SET payload=json_set(payload,'$.box_message_uid',NEW.uid) WHERE category='mission' AND json_extract(payload,'$.box_message_uid')=OLD.uid;
    UPDATE uid_history SET new_uid=NEW.uid WHERE new_uid=OLD.uid;
    INSERT INTO uid_history(old_uid,new_uid) VALUES (OLD.uid,NEW.uid);
END;

CREATE TABLE IF NOT EXISTS support_usage (
    owner INTEGER NOT NULL REFERENCES accounts(uid),
    borrower INTEGER NOT NULL REFERENCES accounts(uid),
    hero INTEGER NOT NULL,
    day INTEGER NOT NULL,
    uses INTEGER NOT NULL DEFAULT 1,
    claimed INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(owner,borrower,hero,day)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS support_usage_borrower ON support_usage(borrower,day,owner);
CREATE TABLE IF NOT EXISTS support_refresh (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    refreshed INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS world_membership (
    uid INTEGER PRIMARY KEY REFERENCES accounts(uid),
    channel TEXT NOT NULL,
    joined INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS world_membership_channel ON world_membership(channel,uid);
CREATE TABLE IF NOT EXISTS world_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT NOT NULL,
    sender INTEGER NOT NULL REFERENCES accounts(uid),
    kind INTEGER NOT NULL,
    content TEXT NOT NULL,
    created INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS world_messages_recent ON world_messages(channel,id DESC);
CREATE TABLE IF NOT EXISTS guild_wars (
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    war INTEGER NOT NULL,
    state BLOB NOT NULL,
    PRIMARY KEY(guild_id,war)
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS guild_war_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    war INTEGER NOT NULL,
    area INTEGER NOT NULL,
    group_id INTEGER NOT NULL,
    uid INTEGER NOT NULL REFERENCES accounts(uid),
    duration INTEGER NOT NULL,
    record BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS guild_war_history_group ON guild_war_history(guild_id,war,area,group_id,id DESC);

CREATE TABLE IF NOT EXISTS guild_war_progress (
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    war INTEGER NOT NULL,
    uid INTEGER NOT NULL REFERENCES accounts(uid),
    area INTEGER NOT NULL,
    score INTEGER NOT NULL,
    assist INTEGER NOT NULL,
    score_time INTEGER NOT NULL,
    assist_time INTEGER NOT NULL,
    PRIMARY KEY(guild_id,war,uid,area)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS async_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    attacker INTEGER NOT NULL REFERENCES accounts(uid),
    defender INTEGER REFERENCES accounts(uid),
    opponent INTEGER NOT NULL,
    friend INTEGER NOT NULL,
    created INTEGER NOT NULL,
    record TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS async_record_attacker ON async_records(attacker,friend,id DESC);
CREATE INDEX IF NOT EXISTS async_record_defender ON async_records(defender,friend,id DESC);
CREATE TRIGGER IF NOT EXISTS network_uid_changed AFTER UPDATE OF uid ON accounts WHEN NEW.uid != OLD.uid
BEGIN
    UPDATE support_usage SET owner=NEW.uid WHERE owner=OLD.uid;
    UPDATE support_usage SET borrower=NEW.uid WHERE borrower=OLD.uid;
    UPDATE support_refresh SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE world_membership SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE world_messages SET sender=NEW.uid WHERE sender=OLD.uid;
    UPDATE guild_war_history SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE guild_war_progress SET uid=NEW.uid WHERE uid=OLD.uid;
    UPDATE async_records SET attacker=NEW.uid WHERE attacker=OLD.uid;
    UPDATE async_records SET defender=NEW.uid WHERE defender=OLD.uid;
END;

CREATE TABLE IF NOT EXISTS server_clock (
    singleton INTEGER PRIMARY KEY CHECK(singleton=1),
    offset_seconds INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS rank_entries (
    uid INTEGER NOT NULL REFERENCES accounts(uid),
    mode TEXT NOT NULL,
    rank_id INTEGER NOT NULL,
    score INTEGER NOT NULL,
    achieved INTEGER NOT NULL,
    PRIMARY KEY(uid,mode,rank_id)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS rank_entries_order ON rank_entries(mode,rank_id,score DESC,achieved,uid);
CREATE TRIGGER IF NOT EXISTS rank_uid_changed AFTER UPDATE OF uid ON accounts WHEN NEW.uid != OLD.uid
BEGIN
    UPDATE rank_entries SET uid=NEW.uid WHERE uid=OLD.uid;
END;

CREATE TABLE IF NOT EXISTS invite_links (
    invitee INTEGER PRIMARY KEY REFERENCES accounts(uid),
    inviter INTEGER NOT NULL REFERENCES accounts(uid),
    bound INTEGER NOT NULL,
    CHECK(invitee<>inviter)
);
CREATE INDEX IF NOT EXISTS invite_links_inviter ON invite_links(inviter,invitee);
CREATE TRIGGER IF NOT EXISTS invite_uid_changed AFTER UPDATE OF uid ON accounts WHEN NEW.uid != OLD.uid
BEGIN
    UPDATE invite_links SET inviter=NEW.uid WHERE inviter=OLD.uid;
    UPDATE invite_links SET invitee=NEW.uid WHERE invitee=OLD.uid;
END;

CREATE TABLE IF NOT EXISTS guild_contributions (
    guild_id INTEGER NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    uid INTEGER NOT NULL REFERENCES accounts(uid),
    week INTEGER NOT NULL,
    score INTEGER NOT NULL,
    PRIMARY KEY(guild_id,week,uid)
) WITHOUT ROWID;
CREATE TRIGGER IF NOT EXISTS guild_contribution_uid_changed AFTER UPDATE OF uid ON accounts WHEN NEW.uid != OLD.uid
BEGIN
    UPDATE guild_contributions SET uid=NEW.uid WHERE uid=OLD.uid;
END;
`
