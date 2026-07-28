/**
 * The log pane: what the session has done, with timings.
 *
 * It exists so a slow result explains itself — which stage cost what — rather than
 * prompting a guess.
 */

import { useExplorer } from '../../core/store.ts';
import { Empty } from '../components/Field.tsx';

export function LogPane() {
  const log = useExplorer((s) => s.log);
  if (log.length === 0) return <Empty>Nothing logged yet.</Empty>;
  return <pre className="log">{log.join('\n')}</pre>;
}
