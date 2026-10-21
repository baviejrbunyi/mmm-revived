function Battle:CheckAutoPlayCard()
  if self.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
    self.autoPlayCard = true
    self.enableAICard = false
    self.endByTimeType = 0
    return
  end
  if self:IsPvP() then
    local aiPlayer = self:GetPlayer(true)
    if aiPlayer.nextAutoUseCardFrame <= self:GetCurFrame() then
      aiPlayer:AutoPlayCard()
    end
  end
  if self.engineType == BattleConst.EngineType.REPLAY then
    return
  end
  if self.inMapChange ~= 0 then
    return
  end
  if self:IsAuto() then
    self.autoPlayCard = true
    self.enableAICard = true
  end
  while (self.autoPlayCard or self:CheckSwitch(BattleConst.BattleSwitches.FORCE_AUTO)) and self.enableAICard do
    local player = self:GetPlayer(false)
    if player.nextAutoUseCardFrame <= self:GetCurFrame() then
      player:AutoPlayCard()
    end
    break
  end
end
