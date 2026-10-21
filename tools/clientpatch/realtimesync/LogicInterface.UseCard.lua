function LogicInterface.UseCard(battle, cardOid, slot, toid, choice)
  local args = {
    coid = cardOid,
    slot = slot,
    toid = toid,
    choice = choice
  }
  if battle.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
    MC.GameWorld.Avatar:CallServerMethod("duelOperation", BattleConst.BattleOperation.PLAY_CARD, args)
    return
  end
  battle:Operation(nil, BattleConst.BattleOperation.PLAY_CARD, args)
end
