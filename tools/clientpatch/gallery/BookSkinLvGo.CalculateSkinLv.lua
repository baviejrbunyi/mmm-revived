function BookSkinLvGo:CalculateSkinLv()
  local skinDict = MC.GameWorld.Avatar.HeroComp:GetSkinDict() or {}
  local allExp = 0
  local curLv = 0
  local key = next(skinDict)
  while key ~= nil do
    local skinData = skin_res_data.Get(key)
    local skinEquipData = skinData and skin_bonus_equip.Get(skinData.rarity)
    if skinEquipData and skinEquipData.score then
      allExp = allExp + skinEquipData.score
    end
    key = next(skinDict, key)
  end
  local lv = 1
  local data = skin_bonus_collection.Get(lv)
  while data do
    if allExp >= data.score then
      curLv = lv
    end
    lv = lv + 1
    data = skin_bonus_collection.Get(lv)
  end
  local curLvData = skin_bonus_collection.Get(curLv)
  local curScore = curLvData and curLvData.score or 0
  local nextLvData = skin_bonus_collection.Get(curLv + 1)
  if not nextLvData then
    local prevLvData = skin_bonus_collection.Get(curLv - 1)
    local needExp = math.max(curScore - (prevLvData and prevLvData.score or 0), 1)
    return curLv, needExp, needExp
  end
  return curLv, nextLvData.score - curScore, allExp - curScore
end
