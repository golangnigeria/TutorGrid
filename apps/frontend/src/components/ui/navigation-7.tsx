import { useState } from "react";
import { Link } from "react-router-dom";
import {
  BookOpen,
  Gamepad2,
  LogIn,
  Menu,
  PenTool,
  Video,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { TutorGridMark } from "@/components/TutorGridLogo";

/**
 * Unauthenticated floating pill header (Watermelon UI navigation-7 adapted for TutorGrid Circle).
 */
export function Navigation7() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <header className="pointer-events-none fixed inset-x-0 top-0 z-40 w-full pt-4">
      <div className="mx-auto flex max-w-5xl items-center justify-center px-4 sm:px-6">
        <Popover open={isOpen} onOpenChange={setIsOpen}>
          <PopoverAnchor asChild>
            <div className="pointer-events-auto flex h-14 w-full items-center justify-between gap-3 rounded-full border border-[#F3F0E8]/14 bg-[#181F1A]/86 pr-3 pl-3 backdrop-blur-md">
              <div className="flex items-center gap-2.5">
                <PopoverTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 rounded-full text-[#F3F0E8]/80 hover:bg-[#222B24] hover:text-[#F3F0E8]"
                  >
                    {isOpen ? (
                      <X className="size-4.5" />
                    ) : (
                      <Menu className="size-4.5" />
                    )}
                    <span className="sr-only">Toggle platform overview</span>
                  </Button>
                </PopoverTrigger>

                <Link to="/" className="flex items-center gap-2.5">
                  <TutorGridMark className="h-8 w-8 shrink-0" />
                  <div className="leading-tight">
                    <span className="font-display block text-base font-semibold tracking-tight text-[#F3F0E8]">
                      TutorGrid Circle
                    </span>
                    <span className="hidden text-[11px] text-[#A6B0A8] sm:block">
                      Teaching &amp; Learning Studio
                    </span>
                  </div>
                </Link>
              </div>

              <div className="flex items-center gap-2">
                <span className="hidden rounded-full border border-[#D6C29F]/30 bg-[#D6C29F]/10 px-3 py-1 text-xs font-medium text-[#D6C29F] sm:inline-flex">
                  Pre-K – Grade 12
                </span>
                <Link
                  to="/sign-in"
                  className="inline-flex items-center gap-1.5 rounded-full bg-[#8FAF94] px-4 py-2 text-xs font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8]"
                >
                  <LogIn className="h-3.5 w-3.5" />
                  Sign in
                </Link>
              </div>
            </div>
          </PopoverAnchor>

          <PopoverContent
            align="center"
            sideOffset={14}
            className={cn(
              "max-h-[75dvh] w-[calc(100vw-2rem)] max-w-4xl overflow-y-auto rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/94 p-0 text-[#F3F0E8] shadow-none ring-0 backdrop-blur-md",
            )}
          >
            <div className="grid grid-cols-1 divide-y divide-[#F3F0E8]/10 px-6 py-6 md:grid-cols-2 md:divide-x md:divide-y-0 md:px-8">
              <div className="pb-5 md:pr-6 md:pb-0">
                <p className="text-[11px] font-medium tracking-wider text-[#D6C29F] uppercase">
                  Core Engines
                </p>
                <ul className="mt-3 space-y-3 text-sm">
                  <li className="flex items-start gap-2.5">
                    <BookOpen className="mt-0.5 h-4 w-4 shrink-0 text-[#8FAF94]" />
                    <div>
                      <p className="font-medium text-[#F3F0E8]">Lesson Studio</p>
                      <p className="text-xs text-[#A6B0A8]">
                        Structured, versioned lessons and resource attachments.
                      </p>
                    </div>
                  </li>
                  <li className="flex items-start gap-2.5">
                    <PenTool className="mt-0.5 h-4 w-4 shrink-0 text-[#8FAF94]" />
                    <div>
                      <p className="font-medium text-[#F3F0E8]">Whiteboard Engine</p>
                      <p className="text-xs text-[#A6B0A8]">
                        Collaborative teaching canvas with isolated board surfaces.
                      </p>
                    </div>
                  </li>
                </ul>
              </div>

              <div className="pt-5 md:pt-0 md:pl-6">
                <p className="text-[11px] font-medium tracking-wider text-[#D6C29F] uppercase">
                  Delivery
                </p>
                <ul className="mt-3 space-y-3 text-sm">
                  <li className="flex items-start gap-2.5">
                    <Gamepad2 className="mt-0.5 h-4 w-4 shrink-0 text-[#8FAF94]" />
                    <div>
                      <p className="font-medium text-[#F3F0E8]">Activity Engine</p>
                      <p className="text-xs text-[#A6B0A8]">
                        Interactive, safe educational activities across subjects.
                      </p>
                    </div>
                  </li>
                  <li className="flex items-start gap-2.5">
                    <Video className="mt-0.5 h-4 w-4 shrink-0 text-[#8FAF94]" />
                    <div>
                      <p className="font-medium text-[#F3F0E8]">Live Classroom</p>
                      <p className="text-xs text-[#A6B0A8]">
                        Real-time audio, video, and pinned lesson sessions.
                      </p>
                    </div>
                  </li>
                </ul>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </header>
  );
}
