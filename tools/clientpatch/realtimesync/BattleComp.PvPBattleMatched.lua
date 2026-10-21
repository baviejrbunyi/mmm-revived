function BattleComp:PvPBattleMatched(battleId, gid, priUid, playInfo, oppoInfo, extraInfo, environmentId)
  pcall(function()
    MC.GameWorld.mmmDuel = {
      manual = extraInfo ~= nil and extraInfo.mmmManual == true,
      queue = {},
      head = 1,
      horizon = -1
    }
    if not BattleComp.mmmDuelRpc then
      BattleComp.mmmDuelRpc = true
      BattleComp.OnDuelOperation = function(comp, frame, uid, oper, args)
        local duel = MC.GameWorld.mmmDuel
        if duel ~= nil and duel.manual then
          duel.queue[#duel.queue + 1] = {
            frame = frame,
            uid = uid,
            oper = oper,
            args = args
          }
        end
      end
      BattleComp.OnDuelHorizon = function(comp, frame)
        local duel = MC.GameWorld.mmmDuel
        if duel ~= nil and duel.manual and frame > duel.horizon then
          duel.horizon = frame
        end
      end
      rpc(BattleComp, "onDuelOperation", "OnDuelOperation", rpc.Int(), rpc.Int(), rpc.Int(), rpc.Dict())
      rpc(BattleComp, "onDuelHorizon", "OnDuelHorizon", rpc.Int())
    end
  end)
  if oppoInfo.uid ~= -1 then
    MC.GameWorld.Avatar.BasicComp:DoUpdateCardInfo(oppoInfo.uid, Const.NameCardType.NAME, function(uid, data)
      MC.Panel.PanelPvpDuel:Show(oppoInfo.uid, function()
        MC.GameWorld.BattleMgrNew:CreateRealTimePvPBattle(battleId, gid, priUid, playInfo, oppoInfo, extraInfo, environmentId)
      end, oppoInfo.aiRandSeed, data)
    end)
  else
    MC.Panel.PanelPvpDuel:Show(oppoInfo.uid, function()
      MC.GameWorld.BattleMgrNew:CreateRealTimePvPBattle(battleId, gid, priUid, playInfo, oppoInfo, extraInfo, environmentId)
    end, oppoInfo.aiRandSeed)
  end
end
