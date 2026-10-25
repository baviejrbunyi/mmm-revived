function PanelDaySignPopup:Show(openList, activityIdList)
  self._activityIdList = activityIdList
  local last = 0
  local key = next(SUB_PANEL_CONFIG)
  while key ~= nil do
    if last < key then
      last = key
    end
    key = next(SUB_PANEL_CONFIG, key)
  end
  for index = 1, last do
    if openList[index] and activityIdList[index] ~= nil then
      table.insert(self._openList, index)
      self._leftCount = self._leftCount + 1
    end
  end
  if #self._openList > 0 then
    self._curShowIdx = 1
    self._curPanelConfig = SUB_PANEL_CONFIG[self._openList[self._curShowIdx]]
  end
  self:ExecuteLoadPanel()
end
