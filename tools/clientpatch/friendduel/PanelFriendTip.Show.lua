function PanelFriendTip:Show(uid, btnList, itemObj, source, layoutLeft, args)
  self._itemObj = itemObj
  self._uid = uid
  self._txtUid.text = uid
  self._source = source or Const.FriendDataSource.None
  self._layoutLeft = layoutLeft
  self._args = args
  local isBlack = MC.GameWorld.Avatar.FriendComp:IsBlack(self._uid)
  self._goBlack:SetActive(isBlack)
  self._goLv:SetActive(not isBlack)
  self:_UpdateInfoData()
  MC.GameWorld.Avatar.BasicComp:DoUpdateCardInfo(uid, Const.NameCardType.NAME, bind(self, self._UpdateData), self)
  if BTN_CONFIG[12] == nil then
    pcall(function()
      local function challenge(panel, mode)
        panel._duelMode = mode
        panel:Show(panel._uid, {16, 17}, panel._itemObj, panel._source, panel._layoutLeft, panel._args)
        panel:LayoutPanel()
      end
      local function send(panel, manual)
        MC.GameWorld.Avatar:CallServerMethod("duelFriend", panel._uid, panel._duelMode or 1, manual)
        panel:Close()
      end
      PanelFriendTip.OnDuelClicked = function(panel)
        panel:Show(panel._uid, {13, 14, 15}, panel._itemObj, panel._source, panel._layoutLeft, panel._args)
        panel:LayoutPanel()
      end
      PanelFriendTip.OnDuelAutoClicked = function(panel)
        send(panel, false)
      end
      PanelFriendTip.OnDuelManualClicked = function(panel)
        send(panel, true)
      end
      PanelFriendTip.OnDuelOwnClicked = function(panel)
        challenge(panel, 1)
      end
      PanelFriendTip.OnDuelRandomClicked = function(panel)
        challenge(panel, 2)
      end
      PanelFriendTip.OnDuelDraftClicked = function(panel)
        challenge(panel, 3)
      end
      PanelFriendTip.CheckDuelDraftLock = function(panel)
        local ok, ready = pcall(function()
          local comp = MC.GameWorld.Avatar.ActivityComp
          local aid = comp:GetPvpActivityId()
          local info = aid and comp:GetActivityData(aid)
          return info ~= nil and info.status == Const.RealtimeRoundStatus.MATCH
        end)
        return not (ok and ready)
      end
      PanelFriendTip.OnDuelDraftLockClicked = function(panel)
        MC.UIUtils.ShowNotify("FRIEND_DUEL_DRAFT_LOCKED")
      end
      BTN_CONFIG[13] = {
        name = "FRIEND_DUEL_MODE_OWN",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_haoyou_mingpian.png",
        func = "OnDuelOwnClicked"
      }
      BTN_CONFIG[14] = {
        name = "FRIEND_DUEL_MODE_RANDOM",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_trans.png",
        func = "OnDuelRandomClicked"
      }
      BTN_CONFIG[15] = {
        name = "FRIEND_DUEL_MODE_DRAFT",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_appointposition.png",
        func = "OnDuelDraftClicked",
        checkLockFunc = "CheckDuelDraftLock",
        lockFunc = "OnDuelDraftLockClicked"
      }
      BTN_CONFIG[16] = {
        name = "FRIEND_DUEL_CONTROL_AUTO",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_trans.png",
        func = "OnDuelAutoClicked"
      }
      BTN_CONFIG[17] = {
        name = "FRIEND_DUEL_CONTROL_MANUAL",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_invite.png",
        func = "OnDuelManualClicked"
      }
      BTN_CONFIG[12] = {
        name = "COMMON_BOTTON_FIGHT",
        icon = "UI/Images/AtlasSprites/TeamInfo/icon_invite.png",
        func = "OnDuelClicked"
      }
    end)
  end
  if btnList == nil then
    btnList = MC.GameWorld.Avatar.FriendComp:GetFriendBtns(uid)
    local friendComp = MC.GameWorld.Avatar.FriendComp
    if BTN_CONFIG[12] ~= nil and uid ~= MC.GameWorld.Avatar.uid and friendComp:IsFriend(uid) and not friendComp:IsBlack(uid) then
      table.insert(btnList, 2, 12)
    end
  end
  self._btnList:RemoveAllItems()
  for index = 1, #btnList do
    local data = BTN_CONFIG[btnList[index]]
    local item = self._btnList:AddItem()
    local isLock = data.checkLockFunc and self[data.checkLockFunc](self) or false
    local lockFunc = data.lockFunc and bind(self, self[data.lockFunc]) or nil
    item:SetInfo(data.icon, data.name, bind(self, self[data.func]), isLock, lockFunc)
  end
end
