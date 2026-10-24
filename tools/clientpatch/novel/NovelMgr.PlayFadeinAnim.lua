function NovelMgr:PlayFadeinAnim(onComplete)
  if not self.vxAnimator then
    return
  end
  local fadeType = self._fadeInType
  self._shownFadeInType = fadeType
  local go, animator, anim
  if fadeType == NovelConst.FadeType.BLACK then
    go, animator, anim = self.vxBlackGo, self.vxBlackAnimator, "com_black_show"
  elseif fadeType == NovelConst.FadeType.SKIP then
    if onComplete then
      mcpcall(onComplete)
    end
    return
  else
    go, animator, anim = self.vxGo, self.vxAnimator, "com_interlude_show"
  end
  go:SetActive(true)
  MC.UIUtils.PlayAnimator(animator, anim, self, function()
    if onComplete then
      onComplete()
    end
  end, true)
end
