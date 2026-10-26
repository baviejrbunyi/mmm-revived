function PanelBattleNew:GetCardItemCount()
  local count = #self._cards
  local battle = self._battle
  local limit = battle and not battle.fixedHandSize and tonumber(battle.heroLimit) or 0
  if count < limit and 1 < count then
    local first = self._cardPosList[1]
    local last = self._cardPosList[count]
    local root = self._goCardRoot.transform
    local card = self._cards[count]:UObj()
    local cancel = self._listCardCancel[count]:UObj()
    for i = count + 1, limit do
      local slot = u3d.GameObject.Instantiate(card, root)
      slot.name = "card_" .. i
      local item = self:AddSimpleChildUI("UI.Common.BattleNew.CardItem", "battle_ui/card_cont/card_" .. i, "_card" .. i)
      item:EnableAnim(true)
      item:SetVisible(false)
      self._cards[i] = item
      local cancelSlot = u3d.GameObject.Instantiate(cancel, root)
      cancelSlot.name = "cancel_" .. i
      local cancelItem = MC.UIUtils.CreateSimpleUI(CardCancel, cancelSlot)
      self:AddDynamicUI(cancelItem, nil, false, nil)
      self._listCardCancel[i] = cancelItem
      self._cardIdxToPos[i] = i
      self._cardPosToIdx[i] = i
    end
    for i = 1, limit do
      local t = (i - 1) / (limit - 1)
      local position = u3d.Vector3(first.x + (last.x - first.x) * t, first.y + (last.y - first.y) * t, first.z + (last.z - first.z) * t)
      self._cardPosList[i] = position
      self._cards[i]:UObj().transform.localPosition = position
      self._listCardCancel[i]:UObj().transform.localPosition = position
    end
  end
  return #self._cards
end
