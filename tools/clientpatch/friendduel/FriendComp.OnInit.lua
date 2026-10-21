function FriendComp:OnInit()
  self._recommendList = {}
  addpropch(self.friendDataDict, "DICT_ADD_ATTR", bind(self, self._OnUpdateFriendDict), self)
  addpropch(self.friendDataDict, "DICT_DEL_ATTR", bind(self, self._OnUpdateFriendDict), self)
  addpropch(self.friendDataDict, "DICT_UPDATE", bind(self, self._OnUpdateFriendDict), self)
  addpropch(self.blacklist, "LIST_INSERT_ITEM", bind(self, self._OnUpdateBlackList), self)
  addpropch(self.blacklist, "LIST_DEL_ITEM", bind(self, self._OnUpdateBlackList), self)
  addpropch(self.applyDict, "DICT_ADD_ATTR", bind(self, self._OnUpdateApplyDict), self)
  addpropch(self._props, "DICT_UPDATE", bind(self, self._OnTimeUpdate), self)
  addpropch(self.supportRewardRecord, "DICT_ADD_ATTR", bind(self, self._OnUpdateSupportRewardRecord), self)
  addpropch(self.supportRewardRecord, "DICT_UPDATE", bind(self, self._OnUpdateSupportRewardRecord), self)
  addpropch(self.recvGift, "DICT_ADD_ATTR", bind(self, self._OnUpdateRecvGift), self)
  addpropch(self.todayCountOfSendGift, "DICT_ADD_ATTR", bind(self, self._OnUpdateSendGift), self)
  self._recentlyChatDict = {}
  self._checkApplyDict = {}
  self.acceptAllList = {}
  self.isAcceptingAll = false
  self.cacheChatMsgs = {}
  self.corDealNewChat = nil
  self.allAcceptedUids = {}
  self.allAcceptCount = 0
  self.sendWorldMsgTime = nil
  self.emojiRowListCount = {}
  self._blockIndex = 1
  MC.EventUtil.AddHandlers(self, {
    [EventDef.AVATAR_READY] = bind(self, self._OnAvatarReady)
  })
  self:RegisterReddot()
  pcall(function()
    local modes = {
      "FRIEND_DUEL_MODE_OWN",
      "FRIEND_DUEL_MODE_RANDOM",
      "FRIEND_DUEL_MODE_DRAFT"
    }
    local results = {
      [0] = "FRIEND_DUEL_SENT",
      "FRIEND_DUEL_DECLINED",
      "FRIEND_DUEL_BUSY",
      "FRIEND_DUEL_OFFLINE",
      "FRIEND_DUEL_EXPIRED",
      "FRIEND_DUEL_NOT_FRIEND",
      "FRIEND_DUEL_NO_DRAFT",
      "FRIEND_DUEL_CANCELLED",
      "FRIEND_DUEL_NO_HEROES"
    }
    FriendComp.OnFriendDuelInvite = function(comp, uid, name, mode, manual)
      if rawget(_G, "MMMAutotest") ~= nil then
        print("[mmmtest] invite", uid, mode, tostring(manual))
        MC.GameWorld.Avatar:CallServerMethod("duelFriendReply", uid, true)
        return
      end
      local function reply(accept)
        comp._duelInviteUid = nil
        comp._duelInviteAccept = nil
        MC.UIManager:ClosePanel("PanelPopUpConfirm")
        MC.GameWorld.Avatar:CallServerMethod("duelFriendReply", uid, accept)
      end
      local function accept()
        reply(true)
      end
      comp._duelInviteUid = uid
      comp._duelInviteAccept = accept
      MC.UIManager:OpenPanel("PanelPopUpConfirm", function(panel)
        panel:SetDescribeText("FRIEND_DUEL_INVITE", {
          name,
          Strings.Get(modes[mode] or modes[1]),
          Strings.Get(manual and "FRIEND_DUEL_CONTROL_MANUAL" or "FRIEND_DUEL_CONTROL_AUTO")
        })
        panel:SetLeftButtonText("FRIEND_DUEL_DECLINE")
        panel:SetRightButtonText("FRIEND_DUEL_ACCEPT")
        panel:SetLeftButton(function()
          reply(false)
        end)
        panel:SetRightButton(accept)
      end)
    end
    FriendComp.OnFriendDuelResult = function(comp, uid, code)
      if comp._duelInviteUid == uid and (code == 3 or code == 4 or code == 7) then
        local panel = MC.UIManager:GetPanel("PanelPopUpConfirm")
        if panel ~= nil and panel._onClickRight == comp._duelInviteAccept then
          MC.UIManager:ClosePanel("PanelPopUpConfirm")
        end
        comp._duelInviteUid = nil
        comp._duelInviteAccept = nil
      end
      local key = results[code]
      if key ~= nil then
        MC.UIUtils.ShowNotify(key)
      end
    end
    FriendComp.OnFriendPresence = function(comp, uid, online, offline)
      local basic = MC.GameWorld.Avatar.BasicComp
      local cached = basic ~= nil and basic._cacheCard ~= nil and basic._cacheCard[uid] or nil
      if cached ~= nil and cached.data ~= nil then
        cached.data.online = online
        cached.data.offline = offline
      end
      MC.EventUtil.Dispatch(EventDef.FRIEND_DICT_UPDATE)
    end
    rpc(FriendComp, "onFriendDuelInvite", "OnFriendDuelInvite", rpc.Int(), rpc.String(), rpc.Int(), rpc.Bool())
    rpc(FriendComp, "onFriendPresence", "OnFriendPresence", rpc.Int(), rpc.Int(), rpc.Int())
    local args = CS.System.Environment.GetCommandLineArgs()
    local spec
    for i = 0, args.Length - 1 do
      if string.find(args[i], "-mmmAutotest=", 1, true) == 1 then
        spec = string.sub(args[i], 14)
      end
    end
    if spec ~= nil then
      local fields = {}
      local start = 1
      while true do
        local stop = string.find(spec, ":", start, true)
        if stop == nil then
          fields[#fields + 1] = string.sub(spec, start)
          break
        end
        fields[#fields + 1] = string.sub(spec, start, stop - 1)
        start = stop + 1
      end
      local test = {name = fields[1], peer = fields[3], role = fields[4], manual = fields[5] ~= "auto", ticks = 0, lastDuel = -100, ended = false}
      rawset(_G, "MMMAutotest", test)
      local LI = require("Common.BattleNew.ViewInterface.LogicInterface")
      local function log(...)
        print("[mmmtest]", test.name, ...)
      end
      MC.EventUtil.AddHandlers(self, {
        [EventDef.FRIEND_SEARCH_RESULT] = function(list)
          if list ~= nil and list[1] ~= nil and test.peerUid == nil then
            test.peerUid = list[1]
            log("found peer", test.peerUid)
          end
        end
      })
      local function target(battle, coid, player)
        if player == nil then
          return nil
        end
        local actors = player.actorMgr.actors
        for i = 1, #actors do
          local actor = battle:GetObject(actors[i])
          if actor ~= nil and actor:IsAlive() and LI.IsValidCardTarget(battle, coid, actors[i], false) then
            return actors[i]
          end
        end
        return nil
      end
      local function play(battle)
        if LI.IsBattleEnd(battle) then
          if not test.ended then
            test.ended = true
            log("battle end", LI.GetCurFrame(battle))
          end
          return
        end
        test.ended = false
        local me = battle.players[battle.uid]
        local oppo = battle.players[battle.oppoUid]
        if me == nil then
          return
        end
        local candidates = {}
        local hand = me.cardMgr.cards
        for i = 1, #hand do
          candidates[#candidates + 1] = hand[i]
        end
        local active = me.cardMgr.activeCards
        for i = 1, #active do
          candidates[#candidates + 1] = active[i]
        end
        log("tick frame", LI.GetCurFrame(battle), "mana", me.mana, "hand", #hand, "active", #active)
        local stage = MC.StageManager:GetActiveStage()
        local panel = MC.UIManager:GetPanel("PanelBattleNew")
        if stage ~= nil and stage.HandCardSystem ~= nil and panel ~= nil then
          local hcs = stage.HandCardSystem
          local ucs = stage.UseCardSystem
          log("ui", "interactable", tostring(panel._cardInteractable), "pause", tostring(panel._pause), "force", tostring(panel._forceNonInteractable), "auto", tostring(hcs._autoPlayCardOid), "using", tostring(hcs._usingCard), "select", tostring(ucs ~= nil and ucs._cardInfo ~= nil and ucs._cardInfo.oid or nil), "mana", tostring(hcs._mana), "speedbtn", tostring(panel._btnSpeed ~= nil and panel._btnSpeed.IsVisible ~= nil and panel._btnSpeed:IsVisible()), "timer", tostring(panel._goTime ~= nil and panel._goTime.activeSelf), "countdown", tostring(panel._countdownTargetTime))
          for i = 1, #candidates do
            local coid = candidates[i]
            local idx = hcs._cardToIdx[coid]
            local item = idx ~= nil and panel:GetCardItem(idx) or panel:GetSkillCardItemByOid(coid)
            local raycast = item ~= nil and item._emptyGraphic ~= nil and item._emptyGraphic.raycastTarget
            local visible = item ~= nil and item:IsVisible()
            local hit = "none"
            if item ~= nil then
              local probed, top = pcall(function()
                local system = CS.UnityEngine.EventSystems.EventSystem.current
                local target = item:GetClickGO()
                local camera = MC.UIManager:Root():GetComponent(typeof(CS.UnityEngine.Canvas)).worldCamera
                local point = camera:WorldToScreenPoint(target.transform.position)
                local pointer = CS.UnityEngine.EventSystems.PointerEventData(system)
                pointer.position = CS.UnityEngine.Vector2(point.x, point.y)
                local results = CS.System.Collections.Generic.List(CS.UnityEngine.EventSystems.RaycastResult)()
                system:RaycastAll(pointer, results)
                if results.Count == 0 then
                  return "empty"
                end
                local first = results[0].gameObject
                local own = first == target or first.transform:IsChildOf(target.transform)
                return first.name .. "/" .. tostring(own)
              end)
              hit = tostring(top)
            end
            log("ui card", coid, "idx", tostring(idx), "usable", tostring(hcs._cardUsable[coid]), "reason", tostring(hcs._cardUsableReason[coid]), "hero", tostring(hcs._cardHeroUsable[coid]), "canuse", tostring(hcs:CanUseCard(coid)), "cd", tostring(hcs:_IsCardInCd(coid)), "raycast", tostring(raycast), "visible", tostring(visible), "hit", hit)
          end
        end
        for i = 1, #candidates do
          local coid = candidates[i]
          local card = battle:GetObject(coid)
          if card ~= nil then
            local usable, reason = card:CanUse()
            if usable then
              local toid = target(battle, coid, oppo) or target(battle, coid, me)
              local slots = LI.GetPlayCardSlots(battle, coid) or {}
              local slot = slots[1]
              log("play", coid, card.typeId, "cost", card:GetManaCost(), "slot", tostring(slot), "toid", tostring(toid))
              LI.UseCard(battle, coid, slot, toid)
              return
            else
              log("unusable", coid, card.typeId, tostring(reason))
            end
          end
        end
      end
      local function deploy(battle)
        local me = battle.players[battle.uid]
        if me == nil or me.prepareRound == nil or me.prepareRound < 1 or test.deployed == me.prepareRound then
          return
        end
        local actors = me.actorMgr.actors
        local used = {}
        local hero
        for i = 1, #actors do
          local actor = battle:GetObject(actors[i])
          if actor ~= nil then
            if actor.slot ~= nil then
              used[actor.slot] = true
            elseif hero == nil and not actor.banned then
              hero = actor
            end
          end
        end
        if hero == nil then
          return
        end
        local keys
        if me.isAI then
          keys = {}
          local placed = battle:GetCurMap().mapEngine:GetPlacedEnemy() or {}
          local key = next(placed)
          while key ~= nil do
            keys[#keys + 1] = key
            key = next(placed, key)
          end
          table.sort(keys)
        else
          keys = rawget(LI.GetHeroPlantSlots(battle), "keys") or {}
        end
        local slot
        for i = 1, #keys do
          if not used[keys[i]] then
            slot = keys[i]
            break
          end
        end
        if slot == nil then
          return
        end
        test.deployed = me.prepareRound
        log("deploy round", me.prepareRound, hero.oid, hero.typeId, "slot", slot)
        LI.PrepareHeroChangePos(battle, hero.oid, nil, slot)
        LI.ConfirmRealTimePVPHero(battle, {{heroId = hero.typeId, slot = slot}})
      end
      local function tick()
        test.ticks = test.ticks + 1
        local ok, failure = pcall(function()
          local avatar = MC.GameWorld.Avatar
          local manager = MC.GameWorld.BattleMgrNew
          local battle = manager ~= nil and manager:GetCurrentBattle() or nil
          if battle ~= nil and battle.isRealTimePvP then
            if not battle.prepareFinish then
              if test.ticks % 3 == 0 then
                deploy(battle)
              end
            elseif test.ticks % 2 == 0 then
              play(battle)
            end
            return
          end
          local friends = avatar.FriendComp:GetFriendDataDict()
          if test.role == "host" then
            if test.peerUid == nil then
              if test.ticks % 5 == 1 then
                avatar:CallServerMethod("searchPlayerByName", test.peer)
              end
            elseif friends[test.peerUid] == nil then
              if test.ticks % 10 == 1 then
                log("apply friend", test.peerUid)
                avatar:CallServerMethod("applyForFriend", test.peerUid, Const.FriendDataSource.Search)
              end
            elseif test.ticks - test.lastDuel > 60 then
              test.lastDuel = test.ticks
              log("challenge", test.peerUid)
              avatar:CallServerMethod("duelFriend", test.peerUid, 1, test.manual)
            end
          else
            local applies = avatar.FriendComp:GetApplyDict()
            local uids = {}
            local key = next(applies)
            while key ~= nil do
              uids[#uids + 1] = key
              key = next(applies, key)
            end
            if #uids > 0 and test.ticks % 3 == 0 then
              log("accept friends", #uids)
              avatar:CallServerMethod("acceptAllFriendApply", uids)
            end
          end
        end)
        if not ok then
          log("tick error", tostring(failure))
        end
        UTools.DelayCall(self, 1, tick)
      end
      log("autotest", test.role, test.peer)
      UTools.DelayCall(self, 3, tick)
    end
    rpc(FriendComp, "onFriendDuelResult", "OnFriendDuelResult", rpc.Int(), rpc.Int())
  end)
end
