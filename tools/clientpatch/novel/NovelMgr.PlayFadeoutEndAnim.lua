function NovelMgr:PlayFadeoutEndAnim(onComplete, fadeType)
  if not self.vxAnimator then
    return
  end
  fadeType = fadeType or self._fadeOutType
  local go, animator, anim
  if fadeType == NovelConst.FadeType.BLACK then
    go, animator, anim = self.vxBlackGo, self.vxBlackAnimator, "com_black_hide"
  elseif fadeType == NovelConst.FadeType.SKIP then
    if onComplete then
      mcpcall(onComplete)
    end
    return
  else
    go, animator, anim = self.vxGo, self.vxAnimator, "com_interlude_show_right02"
  end
  go:SetActive(true)
  MC.UIUtils.PlayAnimator(animator, anim, self, function()
    go:SetActive(false)
    if onComplete then
      mcpcall(onComplete)
    end
  end, true)
end
