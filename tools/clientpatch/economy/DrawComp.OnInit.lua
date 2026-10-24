function DrawComp:OnInit()
  self._drawCountCache = nil
  self.isShowUPTips = false
  pcall(function()
    local pools = draw_pool_data.Instance()
    local times = rawget(pools, "_mmmTimes")
    if times == nil then
      times = {}
      local id, pool = next(pools.data)
      while id ~= nil do
        times[id] = {pool.beginTime, pool.endTime}
        id, pool = next(pools.data, id)
      end
      rawset(pools, "_mmmTimes", times)
    end
    local open = self.poolOpen
    local id, original = next(times)
    while id ~= nil do
      local pool = pools.data[id]
      local state = open ~= nil and open[id] or nil
      if state == 1 then
        pool.beginTime, pool.endTime = 946656000, 4070880000
      elseif state == 0 then
        pool.beginTime, pool.endTime = 4070880000, original[2]
      else
        pool.beginTime, pool.endTime = original[1], original[2]
      end
      id, original = next(times, id)
    end
    local pages = config("shop_page_data").Instance()
    local page = pages.data[MC.UIConst.ShopPage.RECHARGE]
    if page ~= nil then
      local saved = rawget(pages, "_mmmRecharge")
      if saved == nil then
        saved = {page.hide, page.end_time}
        rawset(pages, "_mmmRecharge", saved)
      end
      if self.dreamstoneShop == 1 then
        page.hide, page.end_time = nil, nil
      else
        page.hide, page.end_time = saved[1], saved[2]
      end
      require("Utils.ShopUtils").__ShopPageData = nil
    end
  end)
end
