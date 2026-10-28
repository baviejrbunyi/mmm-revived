local function applyHelperDrawRules(component)
  local helperDrawPity = component.helperDrawPity or 0
  local helperFiftyFifty = component.helperFiftyFifty or 0
  local helperSelectCount = component.helperSelectCount or 0
  local strategies = config("draw_strategy_data").Instance()
  local strategyData = rawget(strategies, "_mmmHelperOriginal")
  if strategyData == nil then
    strategyData = {}
    for id, row in pairs(strategies.data) do
      strategyData[id] = {row.num or 0, row.advance_num or 0}
    end
    rawset(strategies, "_mmmHelperOriginal", strategyData)
  end
  for id, row in pairs(strategies.data) do
    local original = strategyData[id]
    if original then
      row.num, row.advance_num = original[1], original[2]
      if helperDrawPity > 0 and original[1] >= 60 and original[1] <= 80 then
        row.num = helperDrawPity
      end
      if original[2] > 0 and helperFiftyFifty ~= 0 then
        row.advance_num = helperFiftyFifty == 1 and 2 or 1
      end
    end
  end
  local function updateSelectionPrices(dataName)
    local shopData = config(dataName).Instance()
    local shopPrices = rawget(shopData, "_mmmHelperOriginalPrices")
    if shopPrices == nil then
      shopPrices = {}
      for id, row in pairs(shopData.data) do
        if tonumber(id) >= 3005 and tonumber(id) <= 3008 then
          shopPrices[id] = row.price and row.price[1]
        end
      end
      rawset(shopData, "_mmmHelperOriginalPrices", shopPrices)
    end
    for id, price in pairs(shopPrices) do
      local row = shopData.data[id]
      if row and row.price and price ~= nil then
        row.price[1] = helperSelectCount > 0 and helperSelectCount or price
      end
    end
  end
  updateSelectionPrices("shop_data_cn")
  require("Utils.ShopUtils").__ShopPageData = nil
end

_G.ApplyHelperDrawRules = applyHelperDrawRules

function DrawComp:OnInit()
  self._drawCountCache = nil
  self.isShowUPTips = false
  pcall(function()
    applyHelperDrawRules(self)
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
