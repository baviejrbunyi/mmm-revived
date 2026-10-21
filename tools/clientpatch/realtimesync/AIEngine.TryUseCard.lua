function AIEngine.TryUseCard(actor)
  local action, detail
  local ret = false
  if actor.battle.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
    return ret, action, detail
  end
  if actor.battle.enableAICard then
    local cardId = actor:GetWaitReadingCardId()
    cardId = cardId or actor:GetNextActionAICardId()
    if cardId then
      local cardInfo = {cardId = cardId, checkCd = nil}
      ret, action, detail = AIEngine.TryDoSkill(actor, cardInfo)
      if ret == true then
        actor:UseAICardById(cardId)
        if cardId == actor.waitReadingCardId then
          actor.waitReadingCardId = nil
        end
        return ret, action, detail
      end
    end
    local cardInfos = actor:GetUsableAICard()
    for i = 1, #cardInfos do
      local cardInfo = cardInfos[i]
      ret, action, detail = AIEngine.TryDoSkill(actor, cardInfo)
      if ret == true then
        actor:UseAICard(cardInfo.index)
        return ret, action, detail
      end
    end
  end
  return ret, action, detail
end
