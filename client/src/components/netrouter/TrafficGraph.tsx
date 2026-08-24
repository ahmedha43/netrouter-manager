/**
 * Design reminder — monitoring visuals are small technical plots, never SaaS analytics cards.
 */
type TrafficGraphProps = {
  label: string;
  color: "blue" | "green";
  value: number;
  unit?: string;
};

const bluePath = "M0 41 L10 38 L20 42 L30 30 L40 35 L50 17 L60 29 L70 22 L80 34 L90 16 L100 25 L110 19 L120 32 L130 15 L140 22 L150 13 L160 24 L170 20 L180 31 L190 13 L200 18 L210 8 L220 16 L230 12 L240 21";
const greenPath = "M0 44 L10 35 L20 40 L30 26 L40 31 L50 28 L60 14 L70 23 L80 19 L90 30 L100 18 L110 25 L120 16 L130 24 L140 10 L150 19 L160 15 L170 28 L180 18 L190 8 L200 14 L210 9 L220 18 L230 7 L240 13";

export function TrafficGraph({ label, color, value, unit = "Mbps" }: TrafficGraphProps) {
  const path = color === "blue" ? bluePath : greenPath;
  return (
    <div className="nr-traffic-plot">
      <div className="nr-plot-heading">
        <span>{label}</span>
        <strong>{value.toFixed(1)} {unit}</strong>
      </div>
      <svg viewBox="0 0 240 54" role="img" aria-label={`${label}: ${value.toFixed(1)} ${unit}`} preserveAspectRatio="none">
        <path className="nr-plot-grid" d="M0 12 H240 M0 27 H240 M0 42 H240" />
        <path className={`nr-plot-line ${color}`} d={path} />
      </svg>
    </div>
  );
}
