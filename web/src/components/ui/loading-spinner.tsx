import { cn } from "@/lib/utils";

const sizes = {
  sm: "h-6 w-auto",
  md: "h-10 w-auto",
  lg: "h-16 w-auto",
} as const;

interface LoadingSpinnerProps {
  size?: keyof typeof sizes;
  label?: string;
  className?: string;
}

export function LoadingSpinner({
  size = "md",
  label = "Loading...",
  className,
}: LoadingSpinnerProps) {
  return (
    <div className={cn("flex flex-col items-center justify-center py-8", className)}>
      <img src="/owl-logo.png" alt="" className={cn("animate-spin", sizes[size])} />
      {label && (
        <p className="mt-2 text-sm text-muted-foreground">{label}</p>
      )}
    </div>
  );
}
