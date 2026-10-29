export function EmptyState({
  title,
  children,
  action,
}: {
  title: string;
  children?: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <div className="empty">
      <div className="ic" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M3 13h18v8H3zM7 13V3h10v10M10 8h4" />
        </svg>
      </div>
      <b>{title}</b>
      {children ? <span className="hint">{children}</span> : null}
      {action}
    </div>
  );
}
