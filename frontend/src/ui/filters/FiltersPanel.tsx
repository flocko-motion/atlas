/**
 * @layer ui
 * @description Filters panel with level, content class/type, encoding, date, and confidence controls.
 * @depends core/hooks, core/actions
 * @must-not Contain business logic. Import from graph/.
 */

import { useMemo } from 'react';
import { DatePickerInput } from '@mantine/dates';
import { useAppStore } from '../../core/hooks';
import {
  setLevelFilter,
  setContentClassFilter,
  setContentTypeFilter,
  setEncodingClassFilter,
  setEncodingFormatFilter,
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

// Group "class/value" keys by the class prefix for rendering.
function groupByPrefix(keys: string[]): Map<string, string[]> {
  const groups = new Map<string, string[]>();
  for (const key of keys) {
    const idx = key.indexOf('/');
    const prefix = idx === -1 ? key : key.slice(0, idx);
    const suffix = idx === -1 ? '' : key.slice(idx + 1);
    if (!groups.has(prefix)) groups.set(prefix, []);
    groups.get(prefix)!.push(suffix);
  }
  return groups;
}

export function FiltersPanel() {
  const filters = useAppStore((s) => s.filters);
  const nodes = useAppStore((s) => s.nodes);

  // Enumerate all variants present in the current data
  const { allClasses, contentTypeGroups, allEncodingClasses, encodingFormatGroups } = useMemo(() => {
    const classes = new Set<string>();
    const types = new Set<string>();
    const encClasses = new Set<string>();
    const encFormats = new Set<string>();
    for (const n of nodes.values()) {
      classes.add(n.contentClass);
      types.add(`${n.contentClass}/${n.contentType}`);
      encClasses.add(n.encodingClass);
      encFormats.add(`${n.encodingClass}/${n.encodingFormat}`);
    }
    return {
      allClasses: Array.from(classes).sort(),
      contentTypeGroups: groupByPrefix(Array.from(types).sort()),
      allEncodingClasses: Array.from(encClasses).sort(),
      encodingFormatGroups: groupByPrefix(Array.from(encFormats).sort()),
    };
  }, [nodes]);

  // Count matching nodes
  const matchCount = useMemo(() => {
    let count = 0;
    for (const node of nodes.values()) {
      if (!filters.levels.has(node.level)) continue;
      if (filters.contentClasses.size > 0 && !filters.contentClasses.has(node.contentClass)) continue;
      const ctKey = `${node.contentClass}/${node.contentType}`;
      if (filters.contentTypes.size > 0 && !filters.contentTypes.has(ctKey)) continue;
      if (filters.encodingClasses.size > 0 && !filters.encodingClasses.has(node.encodingClass)) continue;
      const efKey = `${node.encodingClass}/${node.encodingFormat}`;
      if (filters.encodingFormats.size > 0 && !filters.encodingFormats.has(efKey)) continue;
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

  const toggleContentType = (key: string) => {
    const next = new Set(filters.contentTypes);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setContentTypeFilter(next);
  };

  const setContentTypeGroup = (prefix: string, suffixes: string[], on: boolean) => {
    const next = new Set(filters.contentTypes);
    for (const s of suffixes) {
      const key = s === '' ? prefix : `${prefix}/${s}`;
      if (on) next.add(key);
      else next.delete(key);
    }
    setContentTypeFilter(next);
  };

  const toggleEncodingClass = (cls: string) => {
    const next = new Set(filters.encodingClasses);
    if (next.has(cls)) next.delete(cls);
    else next.add(cls);
    setEncodingClassFilter(next);
  };

  const toggleEncodingFormat = (key: string) => {
    const next = new Set(filters.encodingFormats);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setEncodingFormatFilter(next);
  };

  const setEncodingFormatGroup = (prefix: string, suffixes: string[], on: boolean) => {
    const next = new Set(filters.encodingFormats);
    for (const s of suffixes) {
      const key = s === '' ? prefix : `${prefix}/${s}`;
      if (on) next.add(key);
      else next.delete(key);
    }
    setEncodingFormatFilter(next);
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
        <span className="filters-section-title">Content Type</span>
        {Array.from(contentTypeGroups).map(([prefix, suffixes]) => (
          <div key={prefix} className="filters-subgroup">
            <div className="filters-subgroup-header">
              <span className="filters-subgroup-label">{prefix}</span>
              <button
                className="filters-subgroup-btn"
                onClick={() => setContentTypeGroup(prefix, suffixes, true)}
              >
                all
              </button>
              <button
                className="filters-subgroup-btn"
                onClick={() => setContentTypeGroup(prefix, suffixes, false)}
              >
                none
              </button>
            </div>
            <div className="filters-classes">
              {suffixes.map((s) => {
                const key = s === '' ? prefix : `${prefix}/${s}`;
                return (
                  <button
                    key={key}
                    className={`filters-class-chip${filters.contentTypes.has(key) ? ' active' : ''}`}
                    onClick={() => toggleContentType(key)}
                  >
                    {s || prefix}
                  </button>
                );
              })}
            </div>
          </div>
        ))}
      </div>

      <div className="filters-section">
        <span className="filters-section-title">Encoding Class</span>
        <div className="filters-classes">
          {allEncodingClasses.map((cls) => (
            <button
              key={cls}
              className={`filters-class-chip${filters.encodingClasses.has(cls) ? ' active' : ''}`}
              onClick={() => toggleEncodingClass(cls)}
            >
              {cls}
            </button>
          ))}
        </div>
      </div>

      <div className="filters-section">
        <span className="filters-section-title">Encoding Format</span>
        {Array.from(encodingFormatGroups).map(([prefix, suffixes]) => (
          <div key={prefix} className="filters-subgroup">
            <div className="filters-subgroup-header">
              <span className="filters-subgroup-label">{prefix}</span>
              <button
                className="filters-subgroup-btn"
                onClick={() => setEncodingFormatGroup(prefix, suffixes, true)}
              >
                all
              </button>
              <button
                className="filters-subgroup-btn"
                onClick={() => setEncodingFormatGroup(prefix, suffixes, false)}
              >
                none
              </button>
            </div>
            <div className="filters-classes">
              {suffixes.map((s) => {
                const key = s === '' ? prefix : `${prefix}/${s}`;
                return (
                  <button
                    key={key}
                    className={`filters-class-chip${filters.encodingFormats.has(key) ? ' active' : ''}`}
                    onClick={() => toggleEncodingFormat(key)}
                  >
                    {s || prefix}
                  </button>
                );
              })}
            </div>
          </div>
        ))}
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
