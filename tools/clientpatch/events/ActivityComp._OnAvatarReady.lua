function ActivityComp:_OnAvatarReady()
  pcall(function()
    local function apply(rows, starts, ends)
      local times = rawget(rows, "_mmmTimes")
      if times == nil then
        times = {}
        local id, row = next(rows.data)
        while id ~= nil do
          times[id] = {row.start_time, row.end_time}
          id, row = next(rows.data, id)
        end
        rawset(rows, "_mmmTimes", times)
      end
      local id, original = next(times)
      while id ~= nil do
        local row = rows.data[id]
        local first = starts ~= nil and starts[id] or nil
        if first ~= nil then
          local last = ends ~= nil and ends[id] or 0
          row.start_time = first
          if last ~= 0 then
            row.end_time = last
          else
            row.end_time = nil
          end
        else
          row.start_time, row.end_time = original[1], original[2]
        end
        id, original = next(times, id)
      end
    end
    apply(activity_data.Instance(), self.eventStart, self.eventEnd)
    apply(config("shop_data").Instance(), self.shopStart, self.shopEnd)
    require("Utils.ShopUtils").__ShopPageData = nil
  end)
  self:TryClearActivityReddot()
  self:UpdateNextFreshTime()
  self._hasRequiredCSevenSign = false
  self._hasRequiredCTenSign = false
  self._hasRequiredCFifteenSign = false
  self._hasRequiredSevenSign = false
  self._hasRequiredFSevenSign = false
end
