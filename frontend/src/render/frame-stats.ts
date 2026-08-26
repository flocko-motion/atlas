/**
 * package: render / frame-stats
 * type:    logic
 * job:     turn one sample window's raw frame counts into the status bar's frame metrics
 * limits:  pure arithmetic; no Sigma, no DOM — renderer.ts owns the sampler this feeds
 */

/**
 * frameStats is null wherever a window has nothing to report — an idle canvas truthfully
 * draws zero frames, which reads as a stalled one if shown as a bare 0 rather than "nothing
 * happened".
 */
export function frameStats(
  frames: number,
  elapsedMs: number,
  worstGapMs: number,
): { fps: number | null; frameMs: number | null; stallMs: number | null } {
  return {
    fps: frames > 0 ? (frames / elapsedMs) * 1000 : null,
    frameMs: frames > 0 ? elapsedMs / frames : null,
    stallMs: worstGapMs > 0 ? worstGapMs : null,
  };
}
