function Battle:Init(initData)
  local extra = self.extraBattleInfo
  local limit = extra and tonumber(extra.heroLimit) or 0
  if 0 < limit and limit ~= self.heroLimit then
    if self.heroLimit < limit then
      local map = self.mapMgr and self.mapMgr.curMap
      local area = map and map.mapEngine and map.mapEngine.heroBornArea
      if area then
        local slots = 0
        local slot = next(area)
        while slot ~= nil do
          slots = slots + 1
          slot = next(area, slot)
        end
        if slots < limit then
          limit = math.max(slots, self.heroLimit)
        end
      end
    end
    self.heroLimit = limit
  end
  BattleInitBuffsHelper.InitEnvironmentBuffs(self)
  BattleInitBuffsHelper.InitModuleBuffs(self)
  self.recorder = qnew("Common.BattleNew.Recorder.Recorder", self)
  self:AddChild(self.recorder)
  self:InitRandomMgr(0)
  local ret = true
  if self.clientReplay then
    self:ClientReplayInit(initData)
  elseif self:IsNormal() or self:IsAuto() then
    self:NormalInit(initData)
  elseif self:IsReplay() then
    ret = self:ReplayInit(initData)
  elseif self:IsServerMode() then
    RealTimePvPHelper.ServerModeInit(self, initData)
  end
  BattleInitBuffsHelper.AddPlayerEnvironmentBuffs(self)
  BattleAttrHelper.InitBattleDamageRatio(self)
  BattleInitBuffsHelper.InitMapBuffs(self)
  if self:IsReplay() then
    self.finishDeploy = true
    self.replayMgr:PlantInitActors()
  end
  return ret
end
