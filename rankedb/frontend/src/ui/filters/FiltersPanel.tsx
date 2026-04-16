/**
 * @layer ui
 * @description Filters panel with level, content class, date range, and confidence controls.
 * @depends core/hooks, core/actions
 * @must-not Contain business logic. Import from graph/.
 */

import { useMemo } from 'react';
import { DatePickerInput } from '@mantine/dates';
import { useAppStore } from '../../core/hooks';
import {
  setLevelFilter,
  setContentClassFilter,
  setDateRange,
  setMinConfidence,
  temporalPosition,
} from '../../core/actions';
import './FiltersPanel.css';

const LEVEL_COLORS: Record<number, string> = {
  0: 'var(--l0-color)',
  1: 'var(--l1-color)',
  2: 'var(--l2-entity-color)',
};

export function FiltersPanel() {
  const filters = useAppStore((s) => s.filters);
  const nodes = useAppStore((s) => s.nodes);

  // Collect all content classes from data
  const allClasses = useMemo(() => {
    const classes = new Set<string>();
    for (const n of nodes.values()) {
      classes.add(n.contentClass);
    }
    return Array.from(classes).sort();
  }, [nodes]);

  // Count matching nodes
  const matchCount = useMemo(() => {
    let count = 0;
    for (const node of nodes.values()) {
      if (!filters.levels.has(node.level)) continue;
      if (filters.contentClasses.size > 0 && !filters.contentClasses.has(node.contentClass)) continue;
      if (filters.dateRange.from || filters.dateRange.to) {
        const d = temporalPosition(node);
        if (filters.dateRange.from && d < filters.dateRange.from) continue;
        if (filters.dateRange.to && d > filters.dateRange.to) continue;
      }
      if (node.confidence !== null && node.confidence < filters.minConfidence) continue;
      count++;
    }
    return count;
  }, [nodes, filters]);

  const toggleLevel = (level: number) => {
    const next = new Set(filters.levels);
    if (next.has(level)) next.delete(level);
    else next.add(level);
    setLevelFilter(next);
  };

  const toggleClass = (cls: string) => {
    const next = new Set(filters.contentClasses);
    if (next.has(cls)) next.delete(cls);
    else next.add(cls);
    setContentClassFilter(next);
  };

  return (
    <div className="filters-panel">
      <div className="filters-section">
        <span className="filters-section-title">Levels</span>
        <div className="filters-levels">
          {[0, 1, 2].map((level) => (
            <label key={level} className="filters-level-check">
              <input
                type="checkbox"
                checked={filters.levels.has(level)}
                onChange={() => toggleLevel(level)}
              />
              <span
                className="filters-level-dot"
                style={{ background: LEVEL_COLORS[level] }}
              />
              L{level}
            </label>
          ))}
        </div>
      </div>

      <div className="filters-section">
        <span className="filters-section-title">Content Class</span>
        <div className="filters-classes">
          {allClasses.map((cls) => (
            <button
              key={cls}
              className={`filters-class-chip${filters.contentClasses.has(cls) ? ' active' : ''}`}
              onClick={() => toggleClass(cls)}
            >
              {cls}
            </button>
          ))}
        </div>
      </div>

      <div className="filters-section">
        <span className="filters-section-title">Date Range</span>
        <DatePickerInput
          type="range"
          placeholder="Pick date range"
          value={[filters.dateRange.from ?? null, filters.dateRange.to ?? null]}
          onChange={([from, to]) => {
            const toDate = (v: unknown): Date | null => {
              if (v instanceof Date) return v;
              if (typeof v === 'string' && v) return new Date(v);
              return null;
            };
            setDateRange(toDate(from), toDate(to));
          }}
          clearable
          size="xs"
        />
      </div>

      <div className="filters-section">
        <span className="filters-section-title">Min Confidence</span>
        <div className="filters-confidence">
          <input
            type="range"
            min={-100}
            max={100}
            value={Math.round(filters.minConfidence * 100)}
            onChange={(e) => setMinConfidence(Number(e.target.value) / 100)}
          />
          <span className="filters-confidence-value">
            {filters.minConfidence.toFixed(2)}
          </span>
        </div>
      </div>

      <div className="filters-count">
        {matchCount} of {nodes.size} nodes match
      </div>
    </div>
  );
}
