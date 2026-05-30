/* icons.jsx — minimal Material-style line icons (no icon font dep).
   Exports: Icon (to window). Usage: <Icon n="send" s={20} c="var(--blue)" /> */
const ICONS = {
  menu:      'M3 6h18M3 12h18M3 18h18',
  back:      'M19 12H5M12 19l-7-7 7-7',
  more:      'M12 5.5a1 1 0 100-2 1 1 0 000 2zM12 13a1 1 0 100-2 1 1 0 000 2zM12 20.5a1 1 0 100-2 1 1 0 000 2z',
  send:      'M3 11l18-8-8 18-2-7-8-3z',
  add:       'M12 5v14M5 12h14',
  chevron:   'M9 6l6 6-6 6',
  chevdown:  'M6 9l6 6 6-6',
  search:    'M11 18a7 7 0 100-14 7 7 0 000 14zM21 21l-4-4',
  close:     'M6 6l12 12M18 6L6 18',
  attach:    'M21 11l-9 9a5 5 0 01-7-7l9-9a3 3 0 014 4l-9 9a1 1 0 01-2-2l8-8',
  mic:       'M12 15a3 3 0 003-3V6a3 3 0 00-6 0v6a3 3 0 003 3zM5 11a7 7 0 0014 0M12 18v3',
  check:     'M5 12l5 5L20 6',
  info:      'M12 21a9 9 0 100-18 9 9 0 000 18zM12 11v5M12 8h.01',
  chat:      'M21 12a8 8 0 01-11.6 7.1L3 21l1.9-6.4A8 8 0 1121 12z',
  tasks:     'M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01',
  layers:    'M12 3l9 5-9 5-9-5 9-5zM3 13l9 5 9-5',
  spark:     'M12 3l2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6z',
  diff:      'M5 4v16M19 4v16M5 8h14M5 16h14',
  bolt:      'M13 2L4 14h7l-1 8 9-12h-7l1-6z',
  file:      'M6 2h8l4 4v16H6zM14 2v4h4',
  eye:       'M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7zM12 15a3 3 0 100-6 3 3 0 000 6z',
};
function Icon({ n, s = 22, c = 'currentColor', sw = 1.8, fill = 'none' }) {
  return (
    <svg width={s} height={s} viewBox="0 0 24 24" fill={fill} stroke={fill === 'none' ? c : 'none'}
      strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0, display: 'block' }}>
      <path d={ICONS[n]} fill={fill === 'none' ? 'none' : c} />
    </svg>
  );
}
window.Icon = Icon;
