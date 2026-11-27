interface TutorGridLogoProps {
  className?: string;
}

/**
 * Compact interlocking Muted Sage (#8FAF94) & Warm Sand (#D6C29F) circle emblem
 * for TutorGrid Circle's Dark Forest-Moss shell.
 */
export function TutorGridMark({ className = "h-8 w-8" }: TutorGridLogoProps) {
  return (
    <svg
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className={className}
    >
      <circle
        cx="24"
        cy="24"
        r="19"
        stroke="#8FAF94"
        strokeWidth="1.8"
        strokeDasharray="88 32"
        strokeDashoffset="10"
      />
      <circle
        cx="24"
        cy="24"
        r="19"
        stroke="#D6C29F"
        strokeWidth="1.8"
        strokeDasharray="82 38"
        strokeDashoffset="72"
        transform="rotate(16 24 24)"
      />
      <path
        d="M16 19.5H32M24 19.5V31.5"
        stroke="#F3F0E8"
        strokeWidth="2.8"
        strokeLinecap="round"
      />
      <circle cx="31.5" cy="30" r="2.2" fill="#D6C29F" />
    </svg>
  );
}

/**
 * Full dark-mode TutorGrid Circle wordmark lockup with interlocking Sage & Warm Sand arcs.
 */
export function TutorGridLockup({ className = "" }: TutorGridLogoProps) {
  return (
    <div className={`inline-flex flex-col items-center select-none ${className}`}>
      <div className="relative flex flex-col items-center px-6 py-4">
        <svg
          viewBox="0 0 220 124"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 h-full w-full"
        >
          <path
            d="M54 44 A58 58 0 0 1 166 44"
            stroke="#8FAF94"
            strokeWidth="1.75"
            strokeLinecap="round"
          />
          <path
            d="M58 46 A55 55 0 0 1 162 46"
            stroke="#D6C29F"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
          <path
            d="M166 80 A58 58 0 0 1 54 80"
            stroke="#8FAF94"
            strokeWidth="1.75"
            strokeLinecap="round"
          />
          <path
            d="M162 78 A55 55 0 0 1 58 78"
            stroke="#D6C29F"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>

        <span className="font-display relative text-2xl font-semibold tracking-[0.06em] text-[#F3F0E8]">
          TUTOR<span className="text-[#8FAF94]">G</span>R<span className="text-[#D6C29F]">I</span>D
        </span>
        <span className="relative mt-0.5 text-[11px] font-medium tracking-[0.42em] text-[#F3F0E8]/80">
          CIRCLE
        </span>
      </div>
    </div>
  );
}
