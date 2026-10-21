function BattleNewStage:_TryDoFrame()
  if LogicInterface.IsBattleEnd(self.battle) then
    self:SetUpdateEnable(false)
    return
  end
  if self.battle.isRealTimePvP then
    local interval = BattleConst.FrameConfig.TIME_PER_FRAME
    local now = CS.UnityEngine.Time.realtimeSinceStartup
    local current = LogicInterface.GetCurFrame(self.battle)
    if self.battle.realtimeStartAt == nil then
      self.battle.realtimeStartAt = now - current * interval
    end
    local tick = CS.UnityEngine.Time.frameCount
    if self._realtimeTick ~= tick then
      self._realtimeTick = tick
      self._realtimeBudget = 90
    end
    local target = math.floor((now - self.battle.realtimeStartAt) / interval)
    local duel = MC.GameWorld.mmmDuel
    local manual = duel ~= nil and duel.manual
    if manual and duel.horizon < target then
      target = duel.horizon
    end
    if self._realtimeBudget > 0 and current < target then
      self._realtimeBudget = self._realtimeBudget - 1
      if manual then
        self._sceneEntity.SceneTaskExecComp:AddTask(function()
          local battle = self.battle
          local frame = LogicInterface.GetCurFrame(battle)
          while duel.head <= #duel.queue and duel.queue[duel.head].frame <= frame do
            local operation = duel.queue[duel.head]
            duel.head = duel.head + 1
            local ok, result = pcall(battle.Operation, battle, operation.uid, operation.oper, operation.args, true)
            if rawget(_G, "MMMAutotest") ~= nil then
              print("[mmmtest] apply", frame, operation.uid, tostring(operation.args.coid), tostring(ok), tostring(result))
            end
          end
          self:_DoOneFrame()
          if rawget(_G, "MMMAutotest") ~= nil then
            local now = LogicInterface.GetCurFrame(battle)
            if now % 60 == 0 then
              local sum = 0
              local other = battle.priUid == battle.uid and battle.oppoUid or battle.uid
              local players = {battle.players[battle.priUid], battle.players[other]}
              for p = 1, 2 do
                local player = players[p]
                if player ~= nil then
                  local actors = player.actorMgr.actors
                  for i = 1, #actors do
                    local actor = battle:GetObject(actors[i])
                    if actor ~= nil then
                      sum = sum + math.floor((actor.hp or 0) * 1000) + (actor.energy or 0) * 7
                    end
                  end
                  sum = sum + (player.mana or 0) * 13 * p
                end
              end
              print("[mmmtest] state", now, sum)
            end
          end
        end, 0)
      else
        self._sceneEntity.SceneTaskExecComp:AddTask(bind(self, self._DoOneFrame), 0)
      end
    end
    return
  end
  local currTime = CS.UnityEngine.Time.time
  if currTime - self.preFrameTime >= BattleConst.FrameConfig.TIME_PER_FRAME then
    self.preFrameTime = self.preFrameTime + BattleConst.FrameConfig.TIME_PER_FRAME
    self._sceneEntity.SceneTaskExecComp:AddTask(bind(self, self._DoOneFrame), 0)
  end
end
