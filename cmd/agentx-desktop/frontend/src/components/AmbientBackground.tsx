import { useEffect, useRef } from "react";

const PARTICLE_COUNT = 30;
const COLORS = [
  "rgba(255, 0, 146, 0.4)",
  "rgba(255, 0, 146, 0.25)",
  "rgba(0, 255, 255, 0.2)",
  "rgba(174, 0, 255, 0.2)",
];

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  r: number;
  color: string;
}

export default function AmbientBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const particlesRef = useRef<Particle[]>([]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const resize = () => {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    };
    resize();
    window.addEventListener("resize", resize);

    // Init particles
    particlesRef.current = Array.from({ length: PARTICLE_COUNT }, () => ({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.3,
      vy: (Math.random() - 0.5) * 0.3,
      r: Math.random() * 2 + 0.5,
      color: COLORS[Math.floor(Math.random() * COLORS.length)],
    }));

    let raf: number;
    const draw = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      for (const p of particlesRef.current) {
        p.x += p.vx;
        p.y += p.vy;
        // Wrap around edges
        if (p.x < 0) p.x = canvas.width;
        if (p.x > canvas.width) p.x = 0;
        if (p.y < 0) p.y = canvas.height;
        if (p.y > canvas.height) p.y = 0;

        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fillStyle = p.color;
        ctx.fill();
      }
      raf = requestAnimationFrame(draw);
    };
    raf = requestAnimationFrame(draw);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
    };
  }, []);

  return (
    <div className="absolute inset-0 pointer-events-none overflow-hidden">
      {/* Perspective grid */}
      <div
        className="absolute inset-0"
        style={{
          perspective: "400px",
          perspectiveOrigin: "50% 50%",
        }}
      >
        <div
          style={{
            position: "absolute",
            inset: "-50%",
            background:
              "linear-gradient(rgba(255,0,146,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,0,146,0.03) 1px, transparent 1px)",
            backgroundSize: "60px 60px",
            transform: "rotateX(45deg)",
            animation: "grid-scroll 8s linear infinite",
          }}
        />
      </div>

      {/* Scanline overlay */}
      <div
        className="absolute inset-0 opacity-[0.03]"
        style={{
          background:
            "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(255,255,255,0.05) 2px, rgba(255,255,255,0.05) 4px)",
          animation: "scanlines 0.5s linear infinite",
        }}
      />

      {/* Floating particles */}
      <canvas ref={canvasRef} className="absolute inset-0" />
    </div>
  );
}
