function BattleNewStage:_GetPreloadPanels()
  if self:isPvpDefenseBattle() then
    return self.PVP_DEFENSE_PRELOAD_PANELS
  elseif self:isRealTimePvPBattle() then
    local duel = MC.GameWorld.mmmDuel
    if duel ~= nil and duel.manual then
      return {
        "PanelBattleNewSelectProxy",
        "PanelBattleNewTips",
        "PanelRealTimePVPStatus",
        "PanelBattleNew",
        "PanelBattleNewPrepare",
        "PanelBattleNewSkillPromptTips",
        "PanelBattleUltimateShot",
        "PanelBattleNewUseCardTips",
        "PanelBattleNewCardDetail",
        "PanelRealTimePVPRoundNotice",
        "PanelRealTimePVPBattleNotice"
      }
    end
    return self.REALTIME_PVP_PRELOAD_PANELS
  elseif self:isFromPVP() or self:isPvpFriendBattle() then
    return self.ASYNC_PVP_PRELOAD_PANELS
  elseif self:isFromManualPvpBattle() then
    return self.HALLOWEEN_PRELOAD_PANELS
  end
  return self.NOMAL_PRELOAD_PANELS
end
