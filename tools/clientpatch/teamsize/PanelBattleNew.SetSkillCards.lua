function PanelBattleNew:SetSkillCards(cards)
  local useCardSystem = self._battleStage.UseCardSystem
  local count = #self._skillCards
  if count < #cards and 1 < count then
    local last = self._skillSlots[count]
    local from = self._skillSlots[count - 1].transform.localPosition
    local to = last.transform.localPosition
    for i = count + 1, #cards do
      local slot = u3d.GameObject.Instantiate(last, last.transform.parent)
      local step = i - count
      slot.name = "skill_" .. i
      slot.transform.localPosition = u3d.Vector3(to.x + (to.x - from.x) * step, to.y + (to.y - from.y) * step, to.z + (to.z - from.z) * step)
      self._skillSlots[i] = slot
      local cancel = MC.UIUtils.CreateSimpleUI(CardCancel, slot.transform:Find("cancel").gameObject)
      self:AddDynamicUI(cancel, nil, false, nil)
      self._listSkillCancel[i] = cancel
      local card = self:AddSimpleChildUI("UI.Common.BattleNew.SkillCardItem", "battle_ui/skill/" .. slot.name .. "/battle_new_card_2", "_skillCard" .. i)
      self._skillCards[i] = card
      self._skillCardPosList[i] = self._skillCardPosList[count]
      self:SetSCItemDown(i)
      card:SetPointerHandler(self)
      card:SetIdx(i)
      card:EnableAnim(true)
    end
  end
  for i = 1, #cards do
    local v = cards[i]
    if useCardSystem:IsUsing(v.oid) then
      useCardSystem:HandleCancel(v.oid)
    end
    local item = self._skillCards[i]
    if item == nil then
      break
    end
    item:SetTypeId(v.typeId)
    item:SetOid(v.oid)
    item:SetVisible(table.get(self._dictShowSkillCard, v.typeId, true))
    item:SetInPublicCd(false)
    item:SetCdProgress(self._skillCds[v.oid] or 0)
  end
  for i = #cards + 1, #self._skillCards do
    local item = self._skillCards[i]
    item:SetOid(nil)
    item:SetVisible(false)
  end
end
