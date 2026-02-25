interface OwlIconProps {
  className?: string;
}

export function OwlIcon({ className }: OwlIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      fill="none"
      className={className}
    >
      {/* Body */}
      <ellipse cx="32" cy="36" rx="20" ry="22" fill="#1E3A5F" />
      {/* Head */}
      <circle cx="32" cy="18" r="14" fill="#1E3A5F" />
      {/* Ears/horns */}
      <polygon points="20,8 18,0 24,6" fill="#1E3A5F" />
      <polygon points="44,8 46,0 40,6" fill="#1E3A5F" />
      {/* Left eye ring */}
      <circle cx="26" cy="18" r="6" fill="#E8EAED" />
      <circle cx="26" cy="18" r="3" fill="#D97706" />
      <circle cx="26" cy="18" r="1.5" fill="#0F1117" />
      {/* Right eye ring */}
      <circle cx="38" cy="18" r="6" fill="#E8EAED" />
      <circle cx="38" cy="18" r="3" fill="#D97706" />
      <circle cx="38" cy="18" r="1.5" fill="#0F1117" />
      {/* Beak */}
      <polygon points="32,22 29,26 35,26" fill="#D97706" />
      {/* Chest pattern */}
      <ellipse cx="32" cy="40" rx="10" ry="12" fill="#253A54" opacity="0.5" />
      {/* Feet */}
      <ellipse cx="26" cy="57" rx="5" ry="2" fill="#D97706" />
      <ellipse cx="38" cy="57" rx="5" ry="2" fill="#D97706" />
    </svg>
  );
}
