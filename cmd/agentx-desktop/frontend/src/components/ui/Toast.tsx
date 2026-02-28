interface Props {
  message: string;
  type: "success" | "error";
}

export function Toast({ message, type }: Props) {
  return (
    <div className="fixed bottom-6 right-6 z-50 animate-in slide-in-from-bottom-4">
      <div
        className={`px-4 py-3 rounded-lg text-sm shadow-lg border ${
          type === "success"
            ? "bg-green-500/20 text-green-400 border-green-500/30"
            : "bg-red-500/20 text-red-400 border-red-500/30"
        }`}
      >
        {message}
      </div>
    </div>
  );
}
