/**
 * package: render / instances
 * type:    adapter
 * job:     hold the two Sigma instances and say which one the reader is looking at
 * limits:  state and accessors only; what is drawn is the renderer's (-> render/renderer),
 *          and where the camera sits is camera's (-> render/camera)
 *
 * Zooming in shows a second Sigma bound to a small graph, and coming back shows the first one
 * again. Everything that measures or moves the picture has to ask which of the two is in front of
 * the reader, so that question lives on its own rather than in whichever file asked it first.
 */

import type Sigma from 'sigma';
import type { DirectedGraph } from 'graphology';

let union: Sigma | null = null;
let lens: Sigma | null = null;
let lensHost: HTMLElement | null = null;

/** unionOf is the instance drawn from the whole graph, which is the one that is always there. */
export function unionOf(): Sigma | null {
  return union;
}

export function lensOf(): Sigma | null {
  return lens;
}

export function hostOf(): HTMLElement | null {
  return lensHost;
}

export function setUnion(instance: Sigma | null): void {
  union = instance;
}

export function setLens(instance: Sigma | null): void {
  lens = instance;
}

export function setHost(host: HTMLElement | null): void {
  lensHost = host;
}

/** lensShowing reports which graph the reader is looking at. */
export function lensShowing(): boolean {
  return lensHost?.style.visibility === 'visible';
}

/** showing is the instance the reader is looking at, which is the one every bound acts on. */
export function showing(): Sigma | null {
  return lensShowing() ? lens : union;
}

/** both is the pair, for the settings that must hold whichever instance comes forward. */
export function both(): (Sigma | null)[] {
  return [union, lens];
}

/** canvasWidth is the drawn width, which the ruler samples across. */
export function canvasWidth(): number {
  return union?.getDimensions().width ?? 0;
}

/** canvasHeight is the other half of what the bound is measured against. */
export function canvasHeight(): number {
  return union?.getDimensions().height ?? 0;
}

/** shownGraph is the graph the reader is looking at, which is what a stretch acts on. */
export function shownGraph(): DirectedGraph | null {
  return (showing()?.getGraph() as DirectedGraph | undefined) ?? null;
}

/** repaint re-uploads whichever graph is showing, after its positions changed. */
export function repaint(): void {
  showing()?.refresh();
}
