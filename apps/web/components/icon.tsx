const paths: Record<string, React.ReactNode> = {
  check: <path d="M20 6 9 17l-5-5" />,
  x: <path d="M18 6 6 18M6 6l12 12" />,
  plus: <path d="M12 5v14M5 12h14" />,
  arrow: <path d="M4 12h16m-7-7 7 7-7 7" />,
  down: <path d="m6 9 6 6 6-6" />,
  search: (
    <>
      <circle cx="11" cy="11" r="7" />
      <path d="m21 21-5-5" />
    </>
  ),
  user: (
    <>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4.4 3.6-8 8-8s8 3.6 8 8" />
    </>
  ),
  users: (
    <>
      <circle cx="9" cy="8" r="4" />
      <path d="M1 21c0-4 3.6-7 8-7s8 3 8 7M17 4a4 4 0 0 1 0 8M23 21c0-3-2-5.5-5-6.5" />
    </>
  ),
  bell: <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10 21h4" />,
  menu: <path d="M3 6h18M3 12h18M3 18h18" />,
  home: <path d="M3 11 12 3l9 8v10H3zM9 21v-8h6v8" />,
  ballot: <path d="M3 13h18v8H3zM7 13V3h10v10M10 8h4" />,
  chart: <path d="M3 3v18h18M7 16v-5M12 16V6M17 16v-8" />,
  lock: <path d="M4 11h16v10H4zM8 11V7a4 4 0 0 1 8 0v4" />,
  alert: <path d="M12 3 2 21h20zM12 10v5M12 18h.01" />,
  info: (
    <>
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-5M12 8h.01" />
    </>
  ),
  cal: <path d="M3 5h18v16H3zM3 10h18M8 2v6M16 2v6" />,
  clock: (
    <>
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v6l4 2" />
    </>
  ),
  file: <path d="M5 2h9l5 5v15H5zM14 2v5h5M8 13h8M8 17h6" />,
  up: <path d="M12 16V4m0 0L7 9m5-5 5 5M4 20h16" />,
  dl: <path d="M12 4v12m0 0-5-5m5 5 5-5M4 20h16" />,
  sliders: <path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6" />,
  out: <path d="M9 21H3V3h6M16 17l5-5-5-5M21 12H9" />,
  eye: (
    <>
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8S1 12 1 12z" />
      <circle cx="12" cy="12" r="3" />
    </>
  ),
  trash: <path d="M3 6h18M8 6V3h8v3M5 6l1 15h12l1-15M10 10v7M14 10v7" />,
  edit: <path d="M4 20h4L20 8l-4-4L4 16zM14 6l4 4" />,
  shield: <path d="M12 2 4 5v7c0 5 3.5 8.5 8 10 4.5-1.5 8-5 8-10V5zM9 12l2 2 4-4" />,
  hist: <path d="M3 12a9 9 0 1 0 3-6.7L3 8M3 3v5h5M12 7v5l4 2" />,
  moon: (
    <>
      <path d="M2 27 11 10l6 10 3-5 10 12z" fill="currentColor" stroke="none" />
      <circle cx="24" cy="8" r="4" fill="currentColor" stroke="none" />
    </>
  ),
};

export type IconName = keyof typeof paths;

export function Icon({
  name,
  size = "md",
  className,
}: {
  name: IconName;
  size?: "s" | "md" | "l";
  className?: string;
}) {
  const sizeClass = size === "md" ? "" : ` ${size}`;
  return (
    <svg className={`i${sizeClass}${className ? ` ${className}` : ""}`} aria-hidden="true">
      {paths[name]}
    </svg>
  );
}
