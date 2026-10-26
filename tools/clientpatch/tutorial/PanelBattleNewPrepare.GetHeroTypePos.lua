function PanelBattleNewPrepare:GetHeroTypePos(heroId)
  for i = 1, self._listHero:GetItemCount() do
    local item = self._listHero:GetItem(i)
    if item:GetHeroTypeId() == heroId then
      local pos = item:GetCareerPos()
      local viewport = self._listHero:GetViewportTrans()
      if pos ~= nil and viewport ~= nil then
        local rect, scroll = viewport.rect, self._srHeroList
        local function outside(world)
          local point = viewport:InverseTransformPoint(world)
          return scroll.horizontal and (point.x < rect.xMin or point.x > rect.xMax) or scroll.vertical and (point.y < rect.yMin or point.y > rect.yMax)
        end
        local corners = CS.System.Array.CreateInstance(typeof(CS.UnityEngine.Vector3), 4)
        item:UObj().transform:GetWorldCorners(corners)
        local hidden = outside(pos)
        for k = 0, 3 do
          hidden = hidden or outside(corners[k])
        end
        if hidden then
          self._listHero:SetBeginItemIdx(math.max(1, i - 1))
          pos = item:GetCareerPos()
        end
      end
      return pos
    end
  end
end
