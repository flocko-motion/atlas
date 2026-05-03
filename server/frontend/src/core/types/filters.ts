/**
 * @layer core
 * @description Filter state types for node/edge filtering.
 * @depends (none)
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

export interface FilterState {
  levels: Set<number>;
  contentClasses: Set<string>;
  contentTypes: Set<string>;        // "contentClass/contentType" keys
  encodingClasses: Set<string>;
  encodingFormats: Set<string>;     // "encodingClass/encodingFormat" keys
  dateRange: { from: Date | null; to: Date | null };
  minConfidence: number;
}
