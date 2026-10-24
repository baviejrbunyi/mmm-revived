function NovelMgr:PlayFadeinEndAnim(onComplete)
  if not self.vxAnimator then
    return
  end
  local fadeType = self._shownFadeInType or self._fadeInType
  self._shownFadeInType = nil
  local go, animator, anim
  if fadeType == NovelConst.FadeType.BLACK then
    go, animator, anim = self.vxBlackGo, self.vxBlackAnimator, "com_black_hide"
  elseif fadeType == NovelConst.FadeType.SKIP then
    if onComplete then
      mcpcall(onComplete)
    end
    return
  else
    go, animator, anim = self.vxGo, self.vxAnimator, "com_interlude_show_left02"
  end
  go:SetActive(true)
  MC.UIUtils.PlayAnimator(animator, anim, self, function()
    go:SetActive(false)
    if onComplete then
      onComplete()
    end
  end, true)
end
