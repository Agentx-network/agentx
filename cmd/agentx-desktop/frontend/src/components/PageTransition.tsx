import { useState, useEffect, useRef } from "react";

interface Props {
  pageKey: string;
  children: React.ReactNode;
}

export default function PageTransition({ pageKey, children }: Props) {
  const [displayKey, setDisplayKey] = useState(pageKey);
  const [phase, setPhase] = useState<"enter" | "exit">("enter");
  const [displayChildren, setDisplayChildren] = useState(children);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    if (pageKey === displayKey) {
      // Same page — update children in place
      setDisplayChildren(children);
      return;
    }

    // Page changed — start exit
    setPhase("exit");
    clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => {
      setDisplayKey(pageKey);
      setDisplayChildren(children);
      setPhase("enter");
    }, 150); // exit duration

    return () => clearTimeout(timeoutRef.current);
  }, [pageKey, children, displayKey]);

  return (
    <div
      className={`h-full flex-1 min-h-0 flex flex-col ${phase === "enter" ? "animate-page-enter" : "animate-page-exit"}`}
      style={{ willChange: "transform" }}
    >
      {displayChildren}
    </div>
  );
}
