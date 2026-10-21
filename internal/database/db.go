package database

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.accountExistsStmt, err = db.PrepareContext(ctx, AccountExists); err != nil {
		return nil, fmt.Errorf("error preparing query AccountExists: %w", err)
	}
	if q.acknowledgeAllMessagesStmt, err = db.PrepareContext(ctx, AcknowledgeAllMessages); err != nil {
		return nil, fmt.Errorf("error preparing query AcknowledgeAllMessages: %w", err)
	}
	if q.acknowledgePairMessagesStmt, err = db.PrepareContext(ctx, AcknowledgePairMessages); err != nil {
		return nil, fmt.Errorf("error preparing query AcknowledgePairMessages: %w", err)
	}
	if q.acknowledgePeerMessagesStmt, err = db.PrepareContext(ctx, AcknowledgePeerMessages); err != nil {
		return nil, fmt.Errorf("error preparing query AcknowledgePeerMessages: %w", err)
	}
	if q.addGuildWarProgressStmt, err = db.PrepareContext(ctx, AddGuildWarProgress); err != nil {
		return nil, fmt.Errorf("error preparing query AddGuildWarProgress: %w", err)
	}
	if q.addSupportUsageStmt, err = db.PrepareContext(ctx, AddSupportUsage); err != nil {
		return nil, fmt.Errorf("error preparing query AddSupportUsage: %w", err)
	}
	if q.blockEitherStmt, err = db.PrepareContext(ctx, BlockEither); err != nil {
		return nil, fmt.Errorf("error preparing query BlockEither: %w", err)
	}
	if q.claimMailStmt, err = db.PrepareContext(ctx, ClaimMail); err != nil {
		return nil, fmt.Errorf("error preparing query ClaimMail: %w", err)
	}
	if q.claimSupportUsageStmt, err = db.PrepareContext(ctx, ClaimSupportUsage); err != nil {
		return nil, fmt.Errorf("error preparing query ClaimSupportUsage: %w", err)
	}
	if q.claimableGiftStmt, err = db.PrepareContext(ctx, ClaimableGift); err != nil {
		return nil, fmt.Errorf("error preparing query ClaimableGift: %w", err)
	}
	if q.countWorldMembersStmt, err = db.PrepareContext(ctx, CountWorldMembers); err != nil {
		return nil, fmt.Errorf("error preparing query CountWorldMembers: %w", err)
	}
	if q.createGuildStmt, err = db.PrepareContext(ctx, CreateGuild); err != nil {
		return nil, fmt.Errorf("error preparing query CreateGuild: %w", err)
	}
	if q.deleteAllClaimedMailStmt, err = db.PrepareContext(ctx, DeleteAllClaimedMail); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteAllClaimedMail: %w", err)
	}
	if q.deleteAllFriendRequestsStmt, err = db.PrepareContext(ctx, DeleteAllFriendRequests); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteAllFriendRequests: %w", err)
	}
	if q.deleteBlockStmt, err = db.PrepareContext(ctx, DeleteBlock); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteBlock: %w", err)
	}
	if q.deleteClaimedMailStmt, err = db.PrepareContext(ctx, DeleteClaimedMail); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteClaimedMail: %w", err)
	}
	if q.deleteFriendPairStmt, err = db.PrepareContext(ctx, DeleteFriendPair); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteFriendPair: %w", err)
	}
	if q.deleteFriendRequestStmt, err = db.PrepareContext(ctx, DeleteFriendRequest); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteFriendRequest: %w", err)
	}
	if q.deleteFriendRequestPairStmt, err = db.PrepareContext(ctx, DeleteFriendRequestPair); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteFriendRequestPair: %w", err)
	}
	if q.deleteGuildStmt, err = db.PrepareContext(ctx, DeleteGuild); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGuild: %w", err)
	}
	if q.deleteGuildApplicationStmt, err = db.PrepareContext(ctx, DeleteGuildApplication); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGuildApplication: %w", err)
	}
	if q.deleteGuildInviteStmt, err = db.PrepareContext(ctx, DeleteGuildInvite); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGuildInvite: %w", err)
	}
	if q.deleteGuildMemberStmt, err = db.PrepareContext(ctx, DeleteGuildMember); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGuildMember: %w", err)
	}
	if q.deletePlayerGuildApplicationsStmt, err = db.PrepareContext(ctx, DeletePlayerGuildApplications); err != nil {
		return nil, fmt.Errorf("error preparing query DeletePlayerGuildApplications: %w", err)
	}
	if q.deletePlayerGuildInvitesStmt, err = db.PrepareContext(ctx, DeletePlayerGuildInvites); err != nil {
		return nil, fmt.Errorf("error preparing query DeletePlayerGuildInvites: %w", err)
	}
	if q.deleteRecordStmt, err = db.PrepareContext(ctx, DeleteRecord); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteRecord: %w", err)
	}
	if q.deliverMailStmt, err = db.PrepareContext(ctx, DeliverMail); err != nil {
		return nil, fmt.Errorf("error preparing query DeliverMail: %w", err)
	}
	if q.findGuildsByNameStmt, err = db.PrepareContext(ctx, FindGuildsByName); err != nil {
		return nil, fmt.Errorf("error preparing query FindGuildsByName: %w", err)
	}
	if q.findPlayerByIDStmt, err = db.PrepareContext(ctx, FindPlayerByID); err != nil {
		return nil, fmt.Errorf("error preparing query FindPlayerByID: %w", err)
	}
	if q.findPlayersByNameStmt, err = db.PrepareContext(ctx, FindPlayersByName); err != nil {
		return nil, fmt.Errorf("error preparing query FindPlayersByName: %w", err)
	}
	if q.friendCountStmt, err = db.PrepareContext(ctx, FriendCount); err != nil {
		return nil, fmt.Errorf("error preparing query FriendCount: %w", err)
	}
	if q.friendRequestExistsStmt, err = db.PrepareContext(ctx, FriendRequestExists); err != nil {
		return nil, fmt.Errorf("error preparing query FriendRequestExists: %w", err)
	}
	if q.friendshipExistsStmt, err = db.PrepareContext(ctx, FriendshipExists); err != nil {
		return nil, fmt.Errorf("error preparing query FriendshipExists: %w", err)
	}
	if q.getAccountStmt, err = db.PrepareContext(ctx, GetAccount); err != nil {
		return nil, fmt.Errorf("error preparing query GetAccount: %w", err)
	}
	if q.getAccountLevelStmt, err = db.PrepareContext(ctx, GetAccountLevel); err != nil {
		return nil, fmt.Errorf("error preparing query GetAccountLevel: %w", err)
	}
	if q.getGuildStmt, err = db.PrepareContext(ctx, GetGuild); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuild: %w", err)
	}
	if q.getGuildApplicationStmt, err = db.PrepareContext(ctx, GetGuildApplication); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuildApplication: %w", err)
	}
	if q.getGuildInviteStmt, err = db.PrepareContext(ctx, GetGuildInvite); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuildInvite: %w", err)
	}
	if q.getGuildMembershipStmt, err = db.PrepareContext(ctx, GetGuildMembership); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuildMembership: %w", err)
	}
	if q.getGuildPlayerStateStmt, err = db.PrepareContext(ctx, GetGuildPlayerState); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuildPlayerState: %w", err)
	}
	if q.getGuildWarStmt, err = db.PrepareContext(ctx, GetGuildWar); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuildWar: %w", err)
	}
	if q.getGuildsStmt, err = db.PrepareContext(ctx, GetGuilds); err != nil {
		return nil, fmt.Errorf("error preparing query GetGuilds: %w", err)
	}
	if q.getInviteProgressStmt, err = db.PrepareContext(ctx, GetInviteProgress); err != nil {
		return nil, fmt.Errorf("error preparing query GetInviteProgress: %w", err)
	}
	if q.getInviterStmt, err = db.PrepareContext(ctx, GetInviter); err != nil {
		return nil, fmt.Errorf("error preparing query GetInviter: %w", err)
	}
	if q.getMailStmt, err = db.PrepareContext(ctx, GetMail); err != nil {
		return nil, fmt.Errorf("error preparing query GetMail: %w", err)
	}
	if q.getMigrationStmt, err = db.PrepareContext(ctx, GetMigration); err != nil {
		return nil, fmt.Errorf("error preparing query GetMigration: %w", err)
	}
	if q.getPublicEquipmentStmt, err = db.PrepareContext(ctx, GetPublicEquipment); err != nil {
		return nil, fmt.Errorf("error preparing query GetPublicEquipment: %w", err)
	}
	if q.getPublicHeroesStmt, err = db.PrepareContext(ctx, GetPublicHeroes); err != nil {
		return nil, fmt.Errorf("error preparing query GetPublicHeroes: %w", err)
	}
	if q.getPublicProfilesStmt, err = db.PrepareContext(ctx, GetPublicProfiles); err != nil {
		return nil, fmt.Errorf("error preparing query GetPublicProfiles: %w", err)
	}
	if q.getRankEntryStmt, err = db.PrepareContext(ctx, GetRankEntry); err != nil {
		return nil, fmt.Errorf("error preparing query GetRankEntry: %w", err)
	}
	if q.getSocialTimesStmt, err = db.PrepareContext(ctx, GetSocialTimes); err != nil {
		return nil, fmt.Errorf("error preparing query GetSocialTimes: %w", err)
	}
	if q.getSupportRefreshStmt, err = db.PrepareContext(ctx, GetSupportRefresh); err != nil {
		return nil, fmt.Errorf("error preparing query GetSupportRefresh: %w", err)
	}
	if q.getWorldMembershipStmt, err = db.PrepareContext(ctx, GetWorldMembership); err != nil {
		return nil, fmt.Errorf("error preparing query GetWorldMembership: %w", err)
	}
	if q.giftExistsStmt, err = db.PrepareContext(ctx, GiftExists); err != nil {
		return nil, fmt.Errorf("error preparing query GiftExists: %w", err)
	}
	if q.guildAdminCountStmt, err = db.PrepareContext(ctx, GuildAdminCount); err != nil {
		return nil, fmt.Errorf("error preparing query GuildAdminCount: %w", err)
	}
	if q.guildNameExistsStmt, err = db.PrepareContext(ctx, GuildNameExists); err != nil {
		return nil, fmt.Errorf("error preparing query GuildNameExists: %w", err)
	}
	if q.insertBlockStmt, err = db.PrepareContext(ctx, InsertBlock); err != nil {
		return nil, fmt.Errorf("error preparing query InsertBlock: %w", err)
	}
	if q.insertFriendPairStmt, err = db.PrepareContext(ctx, InsertFriendPair); err != nil {
		return nil, fmt.Errorf("error preparing query InsertFriendPair: %w", err)
	}
	if q.insertFriendRequestStmt, err = db.PrepareContext(ctx, InsertFriendRequest); err != nil {
		return nil, fmt.Errorf("error preparing query InsertFriendRequest: %w", err)
	}
	if q.insertGiftStmt, err = db.PrepareContext(ctx, InsertGift); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGift: %w", err)
	}
	if q.insertGuildApplicationStmt, err = db.PrepareContext(ctx, InsertGuildApplication); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildApplication: %w", err)
	}
	if q.insertGuildInviteStmt, err = db.PrepareContext(ctx, InsertGuildInvite); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildInvite: %w", err)
	}
	if q.insertGuildMemberStmt, err = db.PrepareContext(ctx, InsertGuildMember); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildMember: %w", err)
	}
	if q.insertGuildMessageStmt, err = db.PrepareContext(ctx, InsertGuildMessage); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildMessage: %w", err)
	}
	if q.insertGuildNewsStmt, err = db.PrepareContext(ctx, InsertGuildNews); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildNews: %w", err)
	}
	if q.insertGuildReportStmt, err = db.PrepareContext(ctx, InsertGuildReport); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGuildReport: %w", err)
	}
	if q.insertInviteStmt, err = db.PrepareContext(ctx, InsertInvite); err != nil {
		return nil, fmt.Errorf("error preparing query InsertInvite: %w", err)
	}
	if q.insertPrivateMessageStmt, err = db.PrepareContext(ctx, InsertPrivateMessage); err != nil {
		return nil, fmt.Errorf("error preparing query InsertPrivateMessage: %w", err)
	}
	if q.listAccountUIDsStmt, err = db.PrepareContext(ctx, ListAccountUIDs); err != nil {
		return nil, fmt.Errorf("error preparing query ListAccountUIDs: %w", err)
	}
	if q.listApplicationsStmt, err = db.PrepareContext(ctx, ListApplications); err != nil {
		return nil, fmt.Errorf("error preparing query ListApplications: %w", err)
	}
	if q.listAsyncCandidatesStmt, err = db.PrepareContext(ctx, ListAsyncCandidates); err != nil {
		return nil, fmt.Errorf("error preparing query ListAsyncCandidates: %w", err)
	}
	if q.listAsyncRecordsStmt, err = db.PrepareContext(ctx, ListAsyncRecords); err != nil {
		return nil, fmt.Errorf("error preparing query ListAsyncRecords: %w", err)
	}
	if q.listBlocksStmt, err = db.PrepareContext(ctx, ListBlocks); err != nil {
		return nil, fmt.Errorf("error preparing query ListBlocks: %w", err)
	}
	if q.listCommunityMessagesStmt, err = db.PrepareContext(ctx, ListCommunityMessages); err != nil {
		return nil, fmt.Errorf("error preparing query ListCommunityMessages: %w", err)
	}
	if q.listFriendRanksStmt, err = db.PrepareContext(ctx, ListFriendRanks); err != nil {
		return nil, fmt.Errorf("error preparing query ListFriendRanks: %w", err)
	}
	if q.listFriendsStmt, err = db.PrepareContext(ctx, ListFriends); err != nil {
		return nil, fmt.Errorf("error preparing query ListFriends: %w", err)
	}
	if q.listGuildApplicationsStmt, err = db.PrepareContext(ctx, ListGuildApplications); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildApplications: %w", err)
	}
	if q.listGuildInvitesStmt, err = db.PrepareContext(ctx, ListGuildInvites); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildInvites: %w", err)
	}
	if q.listGuildMembersStmt, err = db.PrepareContext(ctx, ListGuildMembers); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildMembers: %w", err)
	}
	if q.listGuildMessagesStmt, err = db.PrepareContext(ctx, ListGuildMessages); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildMessages: %w", err)
	}
	if q.listGuildNewsStmt, err = db.PrepareContext(ctx, ListGuildNews); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildNews: %w", err)
	}
	if q.listGuildRanksStmt, err = db.PrepareContext(ctx, ListGuildRanks); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildRanks: %w", err)
	}
	if q.listGuildWarHistoryStmt, err = db.PrepareContext(ctx, ListGuildWarHistory); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildWarHistory: %w", err)
	}
	if q.listGuildWarProgressStmt, err = db.PrepareContext(ctx, ListGuildWarProgress); err != nil {
		return nil, fmt.Errorf("error preparing query ListGuildWarProgress: %w", err)
	}
	if q.listInviteesStmt, err = db.PrepareContext(ctx, ListInvitees); err != nil {
		return nil, fmt.Errorf("error preparing query ListInvitees: %w", err)
	}
	if q.listMailboxStmt, err = db.PrepareContext(ctx, ListMailbox); err != nil {
		return nil, fmt.Errorf("error preparing query ListMailbox: %w", err)
	}
	if q.listPendingMessagesStmt, err = db.PrepareContext(ctx, ListPendingMessages); err != nil {
		return nil, fmt.Errorf("error preparing query ListPendingMessages: %w", err)
	}
	if q.listPendingNetworkAccountsStmt, err = db.PrepareContext(ctx, ListPendingNetworkAccounts); err != nil {
		return nil, fmt.Errorf("error preparing query ListPendingNetworkAccounts: %w", err)
	}
	if q.listPlayerGuildApplicationsStmt, err = db.PrepareContext(ctx, ListPlayerGuildApplications); err != nil {
		return nil, fmt.Errorf("error preparing query ListPlayerGuildApplications: %w", err)
	}
	if q.listPlayerGuildInvitesStmt, err = db.PrepareContext(ctx, ListPlayerGuildInvites); err != nil {
		return nil, fmt.Errorf("error preparing query ListPlayerGuildInvites: %w", err)
	}
	if q.listPlayerRanksStmt, err = db.PrepareContext(ctx, ListPlayerRanks); err != nil {
		return nil, fmt.Errorf("error preparing query ListPlayerRanks: %w", err)
	}
	if q.listReceivedGiftsStmt, err = db.PrepareContext(ctx, ListReceivedGifts); err != nil {
		return nil, fmt.Errorf("error preparing query ListReceivedGifts: %w", err)
	}
	if q.listRecordsStmt, err = db.PrepareContext(ctx, ListRecords); err != nil {
		return nil, fmt.Errorf("error preparing query ListRecords: %w", err)
	}
	if q.listSentGiftsStmt, err = db.PrepareContext(ctx, ListSentGifts); err != nil {
		return nil, fmt.Errorf("error preparing query ListSentGifts: %w", err)
	}
	if q.listSupportCandidatesStmt, err = db.PrepareContext(ctx, ListSupportCandidates); err != nil {
		return nil, fmt.Errorf("error preparing query ListSupportCandidates: %w", err)
	}
	if q.listSupportUsageStmt, err = db.PrepareContext(ctx, ListSupportUsage); err != nil {
		return nil, fmt.Errorf("error preparing query ListSupportUsage: %w", err)
	}
	if q.listWorldMembersStmt, err = db.PrepareContext(ctx, ListWorldMembers); err != nil {
		return nil, fmt.Errorf("error preparing query ListWorldMembers: %w", err)
	}
	if q.listWorldMessagesStmt, err = db.PrepareContext(ctx, ListWorldMessages); err != nil {
		return nil, fmt.Errorf("error preparing query ListWorldMessages: %w", err)
	}
	if q.mailboxCountStmt, err = db.PrepareContext(ctx, MailboxCount); err != nil {
		return nil, fmt.Errorf("error preparing query MailboxCount: %w", err)
	}
	if q.markGiftClaimedStmt, err = db.PrepareContext(ctx, MarkGiftClaimed); err != nil {
		return nil, fmt.Errorf("error preparing query MarkGiftClaimed: %w", err)
	}
	if q.migrateAccountUIDStmt, err = db.PrepareContext(ctx, MigrateAccountUID); err != nil {
		return nil, fmt.Errorf("error preparing query MigrateAccountUID: %w", err)
	}
	if q.nextAccountUIDStmt, err = db.PrepareContext(ctx, NextAccountUID); err != nil {
		return nil, fmt.Errorf("error preparing query NextAccountUID: %w", err)
	}
	if q.openMailStmt, err = db.PrepareContext(ctx, OpenMail); err != nil {
		return nil, fmt.Errorf("error preparing query OpenMail: %w", err)
	}
	if q.playerLevelStmt, err = db.PrepareContext(ctx, PlayerLevel); err != nil {
		return nil, fmt.Errorf("error preparing query PlayerLevel: %w", err)
	}
	if q.pruneWorldMessagesStmt, err = db.PrepareContext(ctx, PruneWorldMessages); err != nil {
		return nil, fmt.Errorf("error preparing query PruneWorldMessages: %w", err)
	}
	if q.rankBoardSizeStmt, err = db.PrepareContext(ctx, RankBoardSize); err != nil {
		return nil, fmt.Errorf("error preparing query RankBoardSize: %w", err)
	}
	if q.rankHigherCountStmt, err = db.PrepareContext(ctx, RankHigherCount); err != nil {
		return nil, fmt.Errorf("error preparing query RankHigherCount: %w", err)
	}
	if q.readClockOffsetStmt, err = db.PrepareContext(ctx, ReadClockOffset); err != nil {
		return nil, fmt.Errorf("error preparing query ReadClockOffset: %w", err)
	}
	if q.receivedGiftCountStmt, err = db.PrepareContext(ctx, ReceivedGiftCount); err != nil {
		return nil, fmt.Errorf("error preparing query ReceivedGiftCount: %w", err)
	}
	if q.recommendGuildsStmt, err = db.PrepareContext(ctx, RecommendGuilds); err != nil {
		return nil, fmt.Errorf("error preparing query RecommendGuilds: %w", err)
	}
	if q.recommendPlayersStmt, err = db.PrepareContext(ctx, RecommendPlayers); err != nil {
		return nil, fmt.Errorf("error preparing query RecommendPlayers: %w", err)
	}
	if q.requestCountStmt, err = db.PrepareContext(ctx, RequestCount); err != nil {
		return nil, fmt.Errorf("error preparing query RequestCount: %w", err)
	}
	if q.resolveLegacyUIDStmt, err = db.PrepareContext(ctx, ResolveLegacyUID); err != nil {
		return nil, fmt.Errorf("error preparing query ResolveLegacyUID: %w", err)
	}
	if q.saveAsyncRecordStmt, err = db.PrepareContext(ctx, SaveAsyncRecord); err != nil {
		return nil, fmt.Errorf("error preparing query SaveAsyncRecord: %w", err)
	}
	if q.saveClockOffsetStmt, err = db.PrepareContext(ctx, SaveClockOffset); err != nil {
		return nil, fmt.Errorf("error preparing query SaveClockOffset: %w", err)
	}
	if q.saveCommunityMessageStmt, err = db.PrepareContext(ctx, SaveCommunityMessage); err != nil {
		return nil, fmt.Errorf("error preparing query SaveCommunityMessage: %w", err)
	}
	if q.saveGuildWarStmt, err = db.PrepareContext(ctx, SaveGuildWar); err != nil {
		return nil, fmt.Errorf("error preparing query SaveGuildWar: %w", err)
	}
	if q.saveGuildWarHistoryStmt, err = db.PrepareContext(ctx, SaveGuildWarHistory); err != nil {
		return nil, fmt.Errorf("error preparing query SaveGuildWarHistory: %w", err)
	}
	if q.saveMigrationStmt, err = db.PrepareContext(ctx, SaveMigration); err != nil {
		return nil, fmt.Errorf("error preparing query SaveMigration: %w", err)
	}
	if q.saveRankEntryStmt, err = db.PrepareContext(ctx, SaveRankEntry); err != nil {
		return nil, fmt.Errorf("error preparing query SaveRankEntry: %w", err)
	}
	if q.saveRecoveredNetworkStmt, err = db.PrepareContext(ctx, SaveRecoveredNetwork); err != nil {
		return nil, fmt.Errorf("error preparing query SaveRecoveredNetwork: %w", err)
	}
	if q.saveWorldMessageStmt, err = db.PrepareContext(ctx, SaveWorldMessage); err != nil {
		return nil, fmt.Errorf("error preparing query SaveWorldMessage: %w", err)
	}
	if q.sentGiftCountStmt, err = db.PrepareContext(ctx, SentGiftCount); err != nil {
		return nil, fmt.Errorf("error preparing query SentGiftCount: %w", err)
	}
	if q.setFriendNicknameStmt, err = db.PrepareContext(ctx, SetFriendNickname); err != nil {
		return nil, fmt.Errorf("error preparing query SetFriendNickname: %w", err)
	}
	if q.setGuildBossStmt, err = db.PrepareContext(ctx, SetGuildBoss); err != nil {
		return nil, fmt.Errorf("error preparing query SetGuildBoss: %w", err)
	}
	if q.setGuildLeftStmt, err = db.PrepareContext(ctx, SetGuildLeft); err != nil {
		return nil, fmt.Errorf("error preparing query SetGuildLeft: %w", err)
	}
	if q.setGuildRoleStmt, err = db.PrepareContext(ctx, SetGuildRole); err != nil {
		return nil, fmt.Errorf("error preparing query SetGuildRole: %w", err)
	}
	if q.setGuildSearchStmt, err = db.PrepareContext(ctx, SetGuildSearch); err != nil {
		return nil, fmt.Errorf("error preparing query SetGuildSearch: %w", err)
	}
	if q.setPresenceOfflineStmt, err = db.PrepareContext(ctx, SetPresenceOffline); err != nil {
		return nil, fmt.Errorf("error preparing query SetPresenceOffline: %w", err)
	}
	if q.setPresenceOnlineStmt, err = db.PrepareContext(ctx, SetPresenceOnline); err != nil {
		return nil, fmt.Errorf("error preparing query SetPresenceOnline: %w", err)
	}
	if q.setRecommendTimeStmt, err = db.PrepareContext(ctx, SetRecommendTime); err != nil {
		return nil, fmt.Errorf("error preparing query SetRecommendTime: %w", err)
	}
	if q.setSearchTimeStmt, err = db.PrepareContext(ctx, SetSearchTime); err != nil {
		return nil, fmt.Errorf("error preparing query SetSearchTime: %w", err)
	}
	if q.setSupportRefreshStmt, err = db.PrepareContext(ctx, SetSupportRefresh); err != nil {
		return nil, fmt.Errorf("error preparing query SetSupportRefresh: %w", err)
	}
	if q.setWorldMembershipStmt, err = db.PrepareContext(ctx, SetWorldMembership); err != nil {
		return nil, fmt.Errorf("error preparing query SetWorldMembership: %w", err)
	}
	if q.supportOwnerUsageCountStmt, err = db.PrepareContext(ctx, SupportOwnerUsageCount); err != nil {
		return nil, fmt.Errorf("error preparing query SupportOwnerUsageCount: %w", err)
	}
	if q.supportUsageCountStmt, err = db.PrepareContext(ctx, SupportUsageCount); err != nil {
		return nil, fmt.Errorf("error preparing query SupportUsageCount: %w", err)
	}
	if q.updateGuildAnnounceStmt, err = db.PrepareContext(ctx, UpdateGuildAnnounce); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateGuildAnnounce: %w", err)
	}
	if q.updateGuildSettingsStmt, err = db.PrepareContext(ctx, UpdateGuildSettings); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateGuildSettings: %w", err)
	}
	if q.upsertAccountStmt, err = db.PrepareContext(ctx, UpsertAccount); err != nil {
		return nil, fmt.Errorf("error preparing query UpsertAccount: %w", err)
	}
	if q.upsertRecordStmt, err = db.PrepareContext(ctx, UpsertRecord); err != nil {
		return nil, fmt.Errorf("error preparing query UpsertRecord: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.accountExistsStmt != nil {
		if cerr := q.accountExistsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing accountExistsStmt: %w", cerr)
		}
	}
	if q.acknowledgeAllMessagesStmt != nil {
		if cerr := q.acknowledgeAllMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing acknowledgeAllMessagesStmt: %w", cerr)
		}
	}
	if q.acknowledgePairMessagesStmt != nil {
		if cerr := q.acknowledgePairMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing acknowledgePairMessagesStmt: %w", cerr)
		}
	}
	if q.acknowledgePeerMessagesStmt != nil {
		if cerr := q.acknowledgePeerMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing acknowledgePeerMessagesStmt: %w", cerr)
		}
	}
	if q.addGuildWarProgressStmt != nil {
		if cerr := q.addGuildWarProgressStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addGuildWarProgressStmt: %w", cerr)
		}
	}
	if q.addSupportUsageStmt != nil {
		if cerr := q.addSupportUsageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addSupportUsageStmt: %w", cerr)
		}
	}
	if q.blockEitherStmt != nil {
		if cerr := q.blockEitherStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing blockEitherStmt: %w", cerr)
		}
	}
	if q.claimMailStmt != nil {
		if cerr := q.claimMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing claimMailStmt: %w", cerr)
		}
	}
	if q.claimSupportUsageStmt != nil {
		if cerr := q.claimSupportUsageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing claimSupportUsageStmt: %w", cerr)
		}
	}
	if q.claimableGiftStmt != nil {
		if cerr := q.claimableGiftStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing claimableGiftStmt: %w", cerr)
		}
	}
	if q.countWorldMembersStmt != nil {
		if cerr := q.countWorldMembersStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countWorldMembersStmt: %w", cerr)
		}
	}
	if q.createGuildStmt != nil {
		if cerr := q.createGuildStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createGuildStmt: %w", cerr)
		}
	}
	if q.deleteAllClaimedMailStmt != nil {
		if cerr := q.deleteAllClaimedMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteAllClaimedMailStmt: %w", cerr)
		}
	}
	if q.deleteAllFriendRequestsStmt != nil {
		if cerr := q.deleteAllFriendRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteAllFriendRequestsStmt: %w", cerr)
		}
	}
	if q.deleteBlockStmt != nil {
		if cerr := q.deleteBlockStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteBlockStmt: %w", cerr)
		}
	}
	if q.deleteClaimedMailStmt != nil {
		if cerr := q.deleteClaimedMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteClaimedMailStmt: %w", cerr)
		}
	}
	if q.deleteFriendPairStmt != nil {
		if cerr := q.deleteFriendPairStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteFriendPairStmt: %w", cerr)
		}
	}
	if q.deleteFriendRequestStmt != nil {
		if cerr := q.deleteFriendRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteFriendRequestStmt: %w", cerr)
		}
	}
	if q.deleteFriendRequestPairStmt != nil {
		if cerr := q.deleteFriendRequestPairStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteFriendRequestPairStmt: %w", cerr)
		}
	}
	if q.deleteGuildStmt != nil {
		if cerr := q.deleteGuildStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGuildStmt: %w", cerr)
		}
	}
	if q.deleteGuildApplicationStmt != nil {
		if cerr := q.deleteGuildApplicationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGuildApplicationStmt: %w", cerr)
		}
	}
	if q.deleteGuildInviteStmt != nil {
		if cerr := q.deleteGuildInviteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGuildInviteStmt: %w", cerr)
		}
	}
	if q.deleteGuildMemberStmt != nil {
		if cerr := q.deleteGuildMemberStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGuildMemberStmt: %w", cerr)
		}
	}
	if q.deletePlayerGuildApplicationsStmt != nil {
		if cerr := q.deletePlayerGuildApplicationsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deletePlayerGuildApplicationsStmt: %w", cerr)
		}
	}
	if q.deletePlayerGuildInvitesStmt != nil {
		if cerr := q.deletePlayerGuildInvitesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deletePlayerGuildInvitesStmt: %w", cerr)
		}
	}
	if q.deleteRecordStmt != nil {
		if cerr := q.deleteRecordStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteRecordStmt: %w", cerr)
		}
	}
	if q.deliverMailStmt != nil {
		if cerr := q.deliverMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deliverMailStmt: %w", cerr)
		}
	}
	if q.findGuildsByNameStmt != nil {
		if cerr := q.findGuildsByNameStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing findGuildsByNameStmt: %w", cerr)
		}
	}
	if q.findPlayerByIDStmt != nil {
		if cerr := q.findPlayerByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing findPlayerByIDStmt: %w", cerr)
		}
	}
	if q.findPlayersByNameStmt != nil {
		if cerr := q.findPlayersByNameStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing findPlayersByNameStmt: %w", cerr)
		}
	}
	if q.friendCountStmt != nil {
		if cerr := q.friendCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendCountStmt: %w", cerr)
		}
	}
	if q.friendRequestExistsStmt != nil {
		if cerr := q.friendRequestExistsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendRequestExistsStmt: %w", cerr)
		}
	}
	if q.friendshipExistsStmt != nil {
		if cerr := q.friendshipExistsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing friendshipExistsStmt: %w", cerr)
		}
	}
	if q.getAccountStmt != nil {
		if cerr := q.getAccountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAccountStmt: %w", cerr)
		}
	}
	if q.getAccountLevelStmt != nil {
		if cerr := q.getAccountLevelStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAccountLevelStmt: %w", cerr)
		}
	}
	if q.getGuildStmt != nil {
		if cerr := q.getGuildStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildStmt: %w", cerr)
		}
	}
	if q.getGuildApplicationStmt != nil {
		if cerr := q.getGuildApplicationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildApplicationStmt: %w", cerr)
		}
	}
	if q.getGuildInviteStmt != nil {
		if cerr := q.getGuildInviteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildInviteStmt: %w", cerr)
		}
	}
	if q.getGuildMembershipStmt != nil {
		if cerr := q.getGuildMembershipStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildMembershipStmt: %w", cerr)
		}
	}
	if q.getGuildPlayerStateStmt != nil {
		if cerr := q.getGuildPlayerStateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildPlayerStateStmt: %w", cerr)
		}
	}
	if q.getGuildWarStmt != nil {
		if cerr := q.getGuildWarStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildWarStmt: %w", cerr)
		}
	}
	if q.getGuildsStmt != nil {
		if cerr := q.getGuildsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGuildsStmt: %w", cerr)
		}
	}
	if q.getInviteProgressStmt != nil {
		if cerr := q.getInviteProgressStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getInviteProgressStmt: %w", cerr)
		}
	}
	if q.getInviterStmt != nil {
		if cerr := q.getInviterStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getInviterStmt: %w", cerr)
		}
	}
	if q.getMailStmt != nil {
		if cerr := q.getMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMailStmt: %w", cerr)
		}
	}
	if q.getMigrationStmt != nil {
		if cerr := q.getMigrationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMigrationStmt: %w", cerr)
		}
	}
	if q.getPublicEquipmentStmt != nil {
		if cerr := q.getPublicEquipmentStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPublicEquipmentStmt: %w", cerr)
		}
	}
	if q.getPublicHeroesStmt != nil {
		if cerr := q.getPublicHeroesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPublicHeroesStmt: %w", cerr)
		}
	}
	if q.getPublicProfilesStmt != nil {
		if cerr := q.getPublicProfilesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPublicProfilesStmt: %w", cerr)
		}
	}
	if q.getRankEntryStmt != nil {
		if cerr := q.getRankEntryStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRankEntryStmt: %w", cerr)
		}
	}
	if q.getSocialTimesStmt != nil {
		if cerr := q.getSocialTimesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getSocialTimesStmt: %w", cerr)
		}
	}
	if q.getSupportRefreshStmt != nil {
		if cerr := q.getSupportRefreshStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getSupportRefreshStmt: %w", cerr)
		}
	}
	if q.getWorldMembershipStmt != nil {
		if cerr := q.getWorldMembershipStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getWorldMembershipStmt: %w", cerr)
		}
	}
	if q.giftExistsStmt != nil {
		if cerr := q.giftExistsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing giftExistsStmt: %w", cerr)
		}
	}
	if q.guildAdminCountStmt != nil {
		if cerr := q.guildAdminCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing guildAdminCountStmt: %w", cerr)
		}
	}
	if q.guildNameExistsStmt != nil {
		if cerr := q.guildNameExistsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing guildNameExistsStmt: %w", cerr)
		}
	}
	if q.insertBlockStmt != nil {
		if cerr := q.insertBlockStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertBlockStmt: %w", cerr)
		}
	}
	if q.insertFriendPairStmt != nil {
		if cerr := q.insertFriendPairStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertFriendPairStmt: %w", cerr)
		}
	}
	if q.insertFriendRequestStmt != nil {
		if cerr := q.insertFriendRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertFriendRequestStmt: %w", cerr)
		}
	}
	if q.insertGiftStmt != nil {
		if cerr := q.insertGiftStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGiftStmt: %w", cerr)
		}
	}
	if q.insertGuildApplicationStmt != nil {
		if cerr := q.insertGuildApplicationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildApplicationStmt: %w", cerr)
		}
	}
	if q.insertGuildInviteStmt != nil {
		if cerr := q.insertGuildInviteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildInviteStmt: %w", cerr)
		}
	}
	if q.insertGuildMemberStmt != nil {
		if cerr := q.insertGuildMemberStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildMemberStmt: %w", cerr)
		}
	}
	if q.insertGuildMessageStmt != nil {
		if cerr := q.insertGuildMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildMessageStmt: %w", cerr)
		}
	}
	if q.insertGuildNewsStmt != nil {
		if cerr := q.insertGuildNewsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildNewsStmt: %w", cerr)
		}
	}
	if q.insertGuildReportStmt != nil {
		if cerr := q.insertGuildReportStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGuildReportStmt: %w", cerr)
		}
	}
	if q.insertInviteStmt != nil {
		if cerr := q.insertInviteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertInviteStmt: %w", cerr)
		}
	}
	if q.insertPrivateMessageStmt != nil {
		if cerr := q.insertPrivateMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertPrivateMessageStmt: %w", cerr)
		}
	}
	if q.listAccountUIDsStmt != nil {
		if cerr := q.listAccountUIDsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listAccountUIDsStmt: %w", cerr)
		}
	}
	if q.listApplicationsStmt != nil {
		if cerr := q.listApplicationsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listApplicationsStmt: %w", cerr)
		}
	}
	if q.listAsyncCandidatesStmt != nil {
		if cerr := q.listAsyncCandidatesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listAsyncCandidatesStmt: %w", cerr)
		}
	}
	if q.listAsyncRecordsStmt != nil {
		if cerr := q.listAsyncRecordsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listAsyncRecordsStmt: %w", cerr)
		}
	}
	if q.listBlocksStmt != nil {
		if cerr := q.listBlocksStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listBlocksStmt: %w", cerr)
		}
	}
	if q.listCommunityMessagesStmt != nil {
		if cerr := q.listCommunityMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listCommunityMessagesStmt: %w", cerr)
		}
	}
	if q.listFriendRanksStmt != nil {
		if cerr := q.listFriendRanksStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listFriendRanksStmt: %w", cerr)
		}
	}
	if q.listFriendsStmt != nil {
		if cerr := q.listFriendsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listFriendsStmt: %w", cerr)
		}
	}
	if q.listGuildApplicationsStmt != nil {
		if cerr := q.listGuildApplicationsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildApplicationsStmt: %w", cerr)
		}
	}
	if q.listGuildInvitesStmt != nil {
		if cerr := q.listGuildInvitesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildInvitesStmt: %w", cerr)
		}
	}
	if q.listGuildMembersStmt != nil {
		if cerr := q.listGuildMembersStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildMembersStmt: %w", cerr)
		}
	}
	if q.listGuildMessagesStmt != nil {
		if cerr := q.listGuildMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildMessagesStmt: %w", cerr)
		}
	}
	if q.listGuildNewsStmt != nil {
		if cerr := q.listGuildNewsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildNewsStmt: %w", cerr)
		}
	}
	if q.listGuildRanksStmt != nil {
		if cerr := q.listGuildRanksStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildRanksStmt: %w", cerr)
		}
	}
	if q.listGuildWarHistoryStmt != nil {
		if cerr := q.listGuildWarHistoryStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildWarHistoryStmt: %w", cerr)
		}
	}
	if q.listGuildWarProgressStmt != nil {
		if cerr := q.listGuildWarProgressStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listGuildWarProgressStmt: %w", cerr)
		}
	}
	if q.listInviteesStmt != nil {
		if cerr := q.listInviteesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listInviteesStmt: %w", cerr)
		}
	}
	if q.listMailboxStmt != nil {
		if cerr := q.listMailboxStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listMailboxStmt: %w", cerr)
		}
	}
	if q.listPendingMessagesStmt != nil {
		if cerr := q.listPendingMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listPendingMessagesStmt: %w", cerr)
		}
	}
	if q.listPendingNetworkAccountsStmt != nil {
		if cerr := q.listPendingNetworkAccountsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listPendingNetworkAccountsStmt: %w", cerr)
		}
	}
	if q.listPlayerGuildApplicationsStmt != nil {
		if cerr := q.listPlayerGuildApplicationsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listPlayerGuildApplicationsStmt: %w", cerr)
		}
	}
	if q.listPlayerGuildInvitesStmt != nil {
		if cerr := q.listPlayerGuildInvitesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listPlayerGuildInvitesStmt: %w", cerr)
		}
	}
	if q.listPlayerRanksStmt != nil {
		if cerr := q.listPlayerRanksStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listPlayerRanksStmt: %w", cerr)
		}
	}
	if q.listReceivedGiftsStmt != nil {
		if cerr := q.listReceivedGiftsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listReceivedGiftsStmt: %w", cerr)
		}
	}
	if q.listRecordsStmt != nil {
		if cerr := q.listRecordsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listRecordsStmt: %w", cerr)
		}
	}
	if q.listSentGiftsStmt != nil {
		if cerr := q.listSentGiftsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listSentGiftsStmt: %w", cerr)
		}
	}
	if q.listSupportCandidatesStmt != nil {
		if cerr := q.listSupportCandidatesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listSupportCandidatesStmt: %w", cerr)
		}
	}
	if q.listSupportUsageStmt != nil {
		if cerr := q.listSupportUsageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listSupportUsageStmt: %w", cerr)
		}
	}
	if q.listWorldMembersStmt != nil {
		if cerr := q.listWorldMembersStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listWorldMembersStmt: %w", cerr)
		}
	}
	if q.listWorldMessagesStmt != nil {
		if cerr := q.listWorldMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listWorldMessagesStmt: %w", cerr)
		}
	}
	if q.mailboxCountStmt != nil {
		if cerr := q.mailboxCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing mailboxCountStmt: %w", cerr)
		}
	}
	if q.markGiftClaimedStmt != nil {
		if cerr := q.markGiftClaimedStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing markGiftClaimedStmt: %w", cerr)
		}
	}
	if q.migrateAccountUIDStmt != nil {
		if cerr := q.migrateAccountUIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing migrateAccountUIDStmt: %w", cerr)
		}
	}
	if q.nextAccountUIDStmt != nil {
		if cerr := q.nextAccountUIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing nextAccountUIDStmt: %w", cerr)
		}
	}
	if q.openMailStmt != nil {
		if cerr := q.openMailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing openMailStmt: %w", cerr)
		}
	}
	if q.playerLevelStmt != nil {
		if cerr := q.playerLevelStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing playerLevelStmt: %w", cerr)
		}
	}
	if q.pruneWorldMessagesStmt != nil {
		if cerr := q.pruneWorldMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing pruneWorldMessagesStmt: %w", cerr)
		}
	}
	if q.rankBoardSizeStmt != nil {
		if cerr := q.rankBoardSizeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing rankBoardSizeStmt: %w", cerr)
		}
	}
	if q.rankHigherCountStmt != nil {
		if cerr := q.rankHigherCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing rankHigherCountStmt: %w", cerr)
		}
	}
	if q.readClockOffsetStmt != nil {
		if cerr := q.readClockOffsetStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing readClockOffsetStmt: %w", cerr)
		}
	}
	if q.receivedGiftCountStmt != nil {
		if cerr := q.receivedGiftCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing receivedGiftCountStmt: %w", cerr)
		}
	}
	if q.recommendGuildsStmt != nil {
		if cerr := q.recommendGuildsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing recommendGuildsStmt: %w", cerr)
		}
	}
	if q.recommendPlayersStmt != nil {
		if cerr := q.recommendPlayersStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing recommendPlayersStmt: %w", cerr)
		}
	}
	if q.requestCountStmt != nil {
		if cerr := q.requestCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing requestCountStmt: %w", cerr)
		}
	}
	if q.resolveLegacyUIDStmt != nil {
		if cerr := q.resolveLegacyUIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing resolveLegacyUIDStmt: %w", cerr)
		}
	}
	if q.saveAsyncRecordStmt != nil {
		if cerr := q.saveAsyncRecordStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveAsyncRecordStmt: %w", cerr)
		}
	}
	if q.saveClockOffsetStmt != nil {
		if cerr := q.saveClockOffsetStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveClockOffsetStmt: %w", cerr)
		}
	}
	if q.saveCommunityMessageStmt != nil {
		if cerr := q.saveCommunityMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveCommunityMessageStmt: %w", cerr)
		}
	}
	if q.saveGuildWarStmt != nil {
		if cerr := q.saveGuildWarStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveGuildWarStmt: %w", cerr)
		}
	}
	if q.saveGuildWarHistoryStmt != nil {
		if cerr := q.saveGuildWarHistoryStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveGuildWarHistoryStmt: %w", cerr)
		}
	}
	if q.saveMigrationStmt != nil {
		if cerr := q.saveMigrationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveMigrationStmt: %w", cerr)
		}
	}
	if q.saveRankEntryStmt != nil {
		if cerr := q.saveRankEntryStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveRankEntryStmt: %w", cerr)
		}
	}
	if q.saveRecoveredNetworkStmt != nil {
		if cerr := q.saveRecoveredNetworkStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveRecoveredNetworkStmt: %w", cerr)
		}
	}
	if q.saveWorldMessageStmt != nil {
		if cerr := q.saveWorldMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing saveWorldMessageStmt: %w", cerr)
		}
	}
	if q.sentGiftCountStmt != nil {
		if cerr := q.sentGiftCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing sentGiftCountStmt: %w", cerr)
		}
	}
	if q.setFriendNicknameStmt != nil {
		if cerr := q.setFriendNicknameStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setFriendNicknameStmt: %w", cerr)
		}
	}
	if q.setGuildBossStmt != nil {
		if cerr := q.setGuildBossStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setGuildBossStmt: %w", cerr)
		}
	}
	if q.setGuildLeftStmt != nil {
		if cerr := q.setGuildLeftStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setGuildLeftStmt: %w", cerr)
		}
	}
	if q.setGuildRoleStmt != nil {
		if cerr := q.setGuildRoleStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setGuildRoleStmt: %w", cerr)
		}
	}
	if q.setGuildSearchStmt != nil {
		if cerr := q.setGuildSearchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setGuildSearchStmt: %w", cerr)
		}
	}
	if q.setPresenceOfflineStmt != nil {
		if cerr := q.setPresenceOfflineStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setPresenceOfflineStmt: %w", cerr)
		}
	}
	if q.setPresenceOnlineStmt != nil {
		if cerr := q.setPresenceOnlineStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setPresenceOnlineStmt: %w", cerr)
		}
	}
	if q.setRecommendTimeStmt != nil {
		if cerr := q.setRecommendTimeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setRecommendTimeStmt: %w", cerr)
		}
	}
	if q.setSearchTimeStmt != nil {
		if cerr := q.setSearchTimeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setSearchTimeStmt: %w", cerr)
		}
	}
	if q.setSupportRefreshStmt != nil {
		if cerr := q.setSupportRefreshStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setSupportRefreshStmt: %w", cerr)
		}
	}
	if q.setWorldMembershipStmt != nil {
		if cerr := q.setWorldMembershipStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setWorldMembershipStmt: %w", cerr)
		}
	}
	if q.supportOwnerUsageCountStmt != nil {
		if cerr := q.supportOwnerUsageCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing supportOwnerUsageCountStmt: %w", cerr)
		}
	}
	if q.supportUsageCountStmt != nil {
		if cerr := q.supportUsageCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing supportUsageCountStmt: %w", cerr)
		}
	}
	if q.updateGuildAnnounceStmt != nil {
		if cerr := q.updateGuildAnnounceStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateGuildAnnounceStmt: %w", cerr)
		}
	}
	if q.updateGuildSettingsStmt != nil {
		if cerr := q.updateGuildSettingsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateGuildSettingsStmt: %w", cerr)
		}
	}
	if q.upsertAccountStmt != nil {
		if cerr := q.upsertAccountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing upsertAccountStmt: %w", cerr)
		}
	}
	if q.upsertRecordStmt != nil {
		if cerr := q.upsertRecordStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing upsertRecordStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...any) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...any) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...any) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                                DBTX
	tx                                *sql.Tx
	accountExistsStmt                 *sql.Stmt
	acknowledgeAllMessagesStmt        *sql.Stmt
	acknowledgePairMessagesStmt       *sql.Stmt
	acknowledgePeerMessagesStmt       *sql.Stmt
	addGuildWarProgressStmt           *sql.Stmt
	addSupportUsageStmt               *sql.Stmt
	blockEitherStmt                   *sql.Stmt
	claimMailStmt                     *sql.Stmt
	claimSupportUsageStmt             *sql.Stmt
	claimableGiftStmt                 *sql.Stmt
	countWorldMembersStmt             *sql.Stmt
	createGuildStmt                   *sql.Stmt
	deleteAllClaimedMailStmt          *sql.Stmt
	deleteAllFriendRequestsStmt       *sql.Stmt
	deleteBlockStmt                   *sql.Stmt
	deleteClaimedMailStmt             *sql.Stmt
	deleteFriendPairStmt              *sql.Stmt
	deleteFriendRequestStmt           *sql.Stmt
	deleteFriendRequestPairStmt       *sql.Stmt
	deleteGuildStmt                   *sql.Stmt
	deleteGuildApplicationStmt        *sql.Stmt
	deleteGuildInviteStmt             *sql.Stmt
	deleteGuildMemberStmt             *sql.Stmt
	deletePlayerGuildApplicationsStmt *sql.Stmt
	deletePlayerGuildInvitesStmt      *sql.Stmt
	deleteRecordStmt                  *sql.Stmt
	deliverMailStmt                   *sql.Stmt
	findGuildsByNameStmt              *sql.Stmt
	findPlayerByIDStmt                *sql.Stmt
	findPlayersByNameStmt             *sql.Stmt
	friendCountStmt                   *sql.Stmt
	friendRequestExistsStmt           *sql.Stmt
	friendshipExistsStmt              *sql.Stmt
	getAccountStmt                    *sql.Stmt
	getAccountLevelStmt               *sql.Stmt
	getGuildStmt                      *sql.Stmt
	getGuildApplicationStmt           *sql.Stmt
	getGuildInviteStmt                *sql.Stmt
	getGuildMembershipStmt            *sql.Stmt
	getGuildPlayerStateStmt           *sql.Stmt
	getGuildWarStmt                   *sql.Stmt
	getGuildsStmt                     *sql.Stmt
	getInviteProgressStmt             *sql.Stmt
	getInviterStmt                    *sql.Stmt
	getMailStmt                       *sql.Stmt
	getMigrationStmt                  *sql.Stmt
	getPublicEquipmentStmt            *sql.Stmt
	getPublicHeroesStmt               *sql.Stmt
	getPublicProfilesStmt             *sql.Stmt
	getRankEntryStmt                  *sql.Stmt
	getSocialTimesStmt                *sql.Stmt
	getSupportRefreshStmt             *sql.Stmt
	getWorldMembershipStmt            *sql.Stmt
	giftExistsStmt                    *sql.Stmt
	guildAdminCountStmt               *sql.Stmt
	guildNameExistsStmt               *sql.Stmt
	insertBlockStmt                   *sql.Stmt
	insertFriendPairStmt              *sql.Stmt
	insertFriendRequestStmt           *sql.Stmt
	insertGiftStmt                    *sql.Stmt
	insertGuildApplicationStmt        *sql.Stmt
	insertGuildInviteStmt             *sql.Stmt
	insertGuildMemberStmt             *sql.Stmt
	insertGuildMessageStmt            *sql.Stmt
	insertGuildNewsStmt               *sql.Stmt
	insertGuildReportStmt             *sql.Stmt
	insertInviteStmt                  *sql.Stmt
	insertPrivateMessageStmt          *sql.Stmt
	listAccountUIDsStmt               *sql.Stmt
	listApplicationsStmt              *sql.Stmt
	listAsyncCandidatesStmt           *sql.Stmt
	listAsyncRecordsStmt              *sql.Stmt
	listBlocksStmt                    *sql.Stmt
	listCommunityMessagesStmt         *sql.Stmt
	listFriendRanksStmt               *sql.Stmt
	listFriendsStmt                   *sql.Stmt
	listGuildApplicationsStmt         *sql.Stmt
	listGuildInvitesStmt              *sql.Stmt
	listGuildMembersStmt              *sql.Stmt
	listGuildMessagesStmt             *sql.Stmt
	listGuildNewsStmt                 *sql.Stmt
	listGuildRanksStmt                *sql.Stmt
	listGuildWarHistoryStmt           *sql.Stmt
	listGuildWarProgressStmt          *sql.Stmt
	listInviteesStmt                  *sql.Stmt
	listMailboxStmt                   *sql.Stmt
	listPendingMessagesStmt           *sql.Stmt
	listPendingNetworkAccountsStmt    *sql.Stmt
	listPlayerGuildApplicationsStmt   *sql.Stmt
	listPlayerGuildInvitesStmt        *sql.Stmt
	listPlayerRanksStmt               *sql.Stmt
	listReceivedGiftsStmt             *sql.Stmt
	listRecordsStmt                   *sql.Stmt
	listSentGiftsStmt                 *sql.Stmt
	listSupportCandidatesStmt         *sql.Stmt
	listSupportUsageStmt              *sql.Stmt
	listWorldMembersStmt              *sql.Stmt
	listWorldMessagesStmt             *sql.Stmt
	mailboxCountStmt                  *sql.Stmt
	markGiftClaimedStmt               *sql.Stmt
	migrateAccountUIDStmt             *sql.Stmt
	nextAccountUIDStmt                *sql.Stmt
	openMailStmt                      *sql.Stmt
	playerLevelStmt                   *sql.Stmt
	pruneWorldMessagesStmt            *sql.Stmt
	rankBoardSizeStmt                 *sql.Stmt
	rankHigherCountStmt               *sql.Stmt
	readClockOffsetStmt               *sql.Stmt
	receivedGiftCountStmt             *sql.Stmt
	recommendGuildsStmt               *sql.Stmt
	recommendPlayersStmt              *sql.Stmt
	requestCountStmt                  *sql.Stmt
	resolveLegacyUIDStmt              *sql.Stmt
	saveAsyncRecordStmt               *sql.Stmt
	saveClockOffsetStmt               *sql.Stmt
	saveCommunityMessageStmt          *sql.Stmt
	saveGuildWarStmt                  *sql.Stmt
	saveGuildWarHistoryStmt           *sql.Stmt
	saveMigrationStmt                 *sql.Stmt
	saveRankEntryStmt                 *sql.Stmt
	saveRecoveredNetworkStmt          *sql.Stmt
	saveWorldMessageStmt              *sql.Stmt
	sentGiftCountStmt                 *sql.Stmt
	setFriendNicknameStmt             *sql.Stmt
	setGuildBossStmt                  *sql.Stmt
	setGuildLeftStmt                  *sql.Stmt
	setGuildRoleStmt                  *sql.Stmt
	setGuildSearchStmt                *sql.Stmt
	setPresenceOfflineStmt            *sql.Stmt
	setPresenceOnlineStmt             *sql.Stmt
	setRecommendTimeStmt              *sql.Stmt
	setSearchTimeStmt                 *sql.Stmt
	setSupportRefreshStmt             *sql.Stmt
	setWorldMembershipStmt            *sql.Stmt
	supportOwnerUsageCountStmt        *sql.Stmt
	supportUsageCountStmt             *sql.Stmt
	updateGuildAnnounceStmt           *sql.Stmt
	updateGuildSettingsStmt           *sql.Stmt
	upsertAccountStmt                 *sql.Stmt
	upsertRecordStmt                  *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                                tx,
		tx:                                tx,
		accountExistsStmt:                 q.accountExistsStmt,
		acknowledgeAllMessagesStmt:        q.acknowledgeAllMessagesStmt,
		acknowledgePairMessagesStmt:       q.acknowledgePairMessagesStmt,
		acknowledgePeerMessagesStmt:       q.acknowledgePeerMessagesStmt,
		addGuildWarProgressStmt:           q.addGuildWarProgressStmt,
		addSupportUsageStmt:               q.addSupportUsageStmt,
		blockEitherStmt:                   q.blockEitherStmt,
		claimMailStmt:                     q.claimMailStmt,
		claimSupportUsageStmt:             q.claimSupportUsageStmt,
		claimableGiftStmt:                 q.claimableGiftStmt,
		countWorldMembersStmt:             q.countWorldMembersStmt,
		createGuildStmt:                   q.createGuildStmt,
		deleteAllClaimedMailStmt:          q.deleteAllClaimedMailStmt,
		deleteAllFriendRequestsStmt:       q.deleteAllFriendRequestsStmt,
		deleteBlockStmt:                   q.deleteBlockStmt,
		deleteClaimedMailStmt:             q.deleteClaimedMailStmt,
		deleteFriendPairStmt:              q.deleteFriendPairStmt,
		deleteFriendRequestStmt:           q.deleteFriendRequestStmt,
		deleteFriendRequestPairStmt:       q.deleteFriendRequestPairStmt,
		deleteGuildStmt:                   q.deleteGuildStmt,
		deleteGuildApplicationStmt:        q.deleteGuildApplicationStmt,
		deleteGuildInviteStmt:             q.deleteGuildInviteStmt,
		deleteGuildMemberStmt:             q.deleteGuildMemberStmt,
		deletePlayerGuildApplicationsStmt: q.deletePlayerGuildApplicationsStmt,
		deletePlayerGuildInvitesStmt:      q.deletePlayerGuildInvitesStmt,
		deleteRecordStmt:                  q.deleteRecordStmt,
		deliverMailStmt:                   q.deliverMailStmt,
		findGuildsByNameStmt:              q.findGuildsByNameStmt,
		findPlayerByIDStmt:                q.findPlayerByIDStmt,
		findPlayersByNameStmt:             q.findPlayersByNameStmt,
		friendCountStmt:                   q.friendCountStmt,
		friendRequestExistsStmt:           q.friendRequestExistsStmt,
		friendshipExistsStmt:              q.friendshipExistsStmt,
		getAccountStmt:                    q.getAccountStmt,
		getAccountLevelStmt:               q.getAccountLevelStmt,
		getGuildStmt:                      q.getGuildStmt,
		getGuildApplicationStmt:           q.getGuildApplicationStmt,
		getGuildInviteStmt:                q.getGuildInviteStmt,
		getGuildMembershipStmt:            q.getGuildMembershipStmt,
		getGuildPlayerStateStmt:           q.getGuildPlayerStateStmt,
		getGuildWarStmt:                   q.getGuildWarStmt,
		getGuildsStmt:                     q.getGuildsStmt,
		getInviteProgressStmt:             q.getInviteProgressStmt,
		getInviterStmt:                    q.getInviterStmt,
		getMailStmt:                       q.getMailStmt,
		getMigrationStmt:                  q.getMigrationStmt,
		getPublicEquipmentStmt:            q.getPublicEquipmentStmt,
		getPublicHeroesStmt:               q.getPublicHeroesStmt,
		getPublicProfilesStmt:             q.getPublicProfilesStmt,
		getRankEntryStmt:                  q.getRankEntryStmt,
		getSocialTimesStmt:                q.getSocialTimesStmt,
		getSupportRefreshStmt:             q.getSupportRefreshStmt,
		getWorldMembershipStmt:            q.getWorldMembershipStmt,
		giftExistsStmt:                    q.giftExistsStmt,
		guildAdminCountStmt:               q.guildAdminCountStmt,
		guildNameExistsStmt:               q.guildNameExistsStmt,
		insertBlockStmt:                   q.insertBlockStmt,
		insertFriendPairStmt:              q.insertFriendPairStmt,
		insertFriendRequestStmt:           q.insertFriendRequestStmt,
		insertGiftStmt:                    q.insertGiftStmt,
		insertGuildApplicationStmt:        q.insertGuildApplicationStmt,
		insertGuildInviteStmt:             q.insertGuildInviteStmt,
		insertGuildMemberStmt:             q.insertGuildMemberStmt,
		insertGuildMessageStmt:            q.insertGuildMessageStmt,
		insertGuildNewsStmt:               q.insertGuildNewsStmt,
		insertGuildReportStmt:             q.insertGuildReportStmt,
		insertInviteStmt:                  q.insertInviteStmt,
		insertPrivateMessageStmt:          q.insertPrivateMessageStmt,
		listAccountUIDsStmt:               q.listAccountUIDsStmt,
		listApplicationsStmt:              q.listApplicationsStmt,
		listAsyncCandidatesStmt:           q.listAsyncCandidatesStmt,
		listAsyncRecordsStmt:              q.listAsyncRecordsStmt,
		listBlocksStmt:                    q.listBlocksStmt,
		listCommunityMessagesStmt:         q.listCommunityMessagesStmt,
		listFriendRanksStmt:               q.listFriendRanksStmt,
		listFriendsStmt:                   q.listFriendsStmt,
		listGuildApplicationsStmt:         q.listGuildApplicationsStmt,
		listGuildInvitesStmt:              q.listGuildInvitesStmt,
		listGuildMembersStmt:              q.listGuildMembersStmt,
		listGuildMessagesStmt:             q.listGuildMessagesStmt,
		listGuildNewsStmt:                 q.listGuildNewsStmt,
		listGuildRanksStmt:                q.listGuildRanksStmt,
		listGuildWarHistoryStmt:           q.listGuildWarHistoryStmt,
		listGuildWarProgressStmt:          q.listGuildWarProgressStmt,
		listInviteesStmt:                  q.listInviteesStmt,
		listMailboxStmt:                   q.listMailboxStmt,
		listPendingMessagesStmt:           q.listPendingMessagesStmt,
		listPendingNetworkAccountsStmt:    q.listPendingNetworkAccountsStmt,
		listPlayerGuildApplicationsStmt:   q.listPlayerGuildApplicationsStmt,
		listPlayerGuildInvitesStmt:        q.listPlayerGuildInvitesStmt,
		listPlayerRanksStmt:               q.listPlayerRanksStmt,
		listReceivedGiftsStmt:             q.listReceivedGiftsStmt,
		listRecordsStmt:                   q.listRecordsStmt,
		listSentGiftsStmt:                 q.listSentGiftsStmt,
		listSupportCandidatesStmt:         q.listSupportCandidatesStmt,
		listSupportUsageStmt:              q.listSupportUsageStmt,
		listWorldMembersStmt:              q.listWorldMembersStmt,
		listWorldMessagesStmt:             q.listWorldMessagesStmt,
		mailboxCountStmt:                  q.mailboxCountStmt,
		markGiftClaimedStmt:               q.markGiftClaimedStmt,
		migrateAccountUIDStmt:             q.migrateAccountUIDStmt,
		nextAccountUIDStmt:                q.nextAccountUIDStmt,
		openMailStmt:                      q.openMailStmt,
		playerLevelStmt:                   q.playerLevelStmt,
		pruneWorldMessagesStmt:            q.pruneWorldMessagesStmt,
		rankBoardSizeStmt:                 q.rankBoardSizeStmt,
		rankHigherCountStmt:               q.rankHigherCountStmt,
		readClockOffsetStmt:               q.readClockOffsetStmt,
		receivedGiftCountStmt:             q.receivedGiftCountStmt,
		recommendGuildsStmt:               q.recommendGuildsStmt,
		recommendPlayersStmt:              q.recommendPlayersStmt,
		requestCountStmt:                  q.requestCountStmt,
		resolveLegacyUIDStmt:              q.resolveLegacyUIDStmt,
		saveAsyncRecordStmt:               q.saveAsyncRecordStmt,
		saveClockOffsetStmt:               q.saveClockOffsetStmt,
		saveCommunityMessageStmt:          q.saveCommunityMessageStmt,
		saveGuildWarStmt:                  q.saveGuildWarStmt,
		saveGuildWarHistoryStmt:           q.saveGuildWarHistoryStmt,
		saveMigrationStmt:                 q.saveMigrationStmt,
		saveRankEntryStmt:                 q.saveRankEntryStmt,
		saveRecoveredNetworkStmt:          q.saveRecoveredNetworkStmt,
		saveWorldMessageStmt:              q.saveWorldMessageStmt,
		sentGiftCountStmt:                 q.sentGiftCountStmt,
		setFriendNicknameStmt:             q.setFriendNicknameStmt,
		setGuildBossStmt:                  q.setGuildBossStmt,
		setGuildLeftStmt:                  q.setGuildLeftStmt,
		setGuildRoleStmt:                  q.setGuildRoleStmt,
		setGuildSearchStmt:                q.setGuildSearchStmt,
		setPresenceOfflineStmt:            q.setPresenceOfflineStmt,
		setPresenceOnlineStmt:             q.setPresenceOnlineStmt,
		setRecommendTimeStmt:              q.setRecommendTimeStmt,
		setSearchTimeStmt:                 q.setSearchTimeStmt,
		setSupportRefreshStmt:             q.setSupportRefreshStmt,
		setWorldMembershipStmt:            q.setWorldMembershipStmt,
		supportOwnerUsageCountStmt:        q.supportOwnerUsageCountStmt,
		supportUsageCountStmt:             q.supportUsageCountStmt,
		updateGuildAnnounceStmt:           q.updateGuildAnnounceStmt,
		updateGuildSettingsStmt:           q.updateGuildSettingsStmt,
		upsertAccountStmt:                 q.upsertAccountStmt,
		upsertRecordStmt:                  q.upsertRecordStmt,
	}
}
