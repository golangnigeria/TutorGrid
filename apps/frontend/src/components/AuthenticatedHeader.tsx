import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Building2,
  Check,
  ChevronDown,
  LogOut,
  ShieldCheck,
} from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { TutorGridMark } from "@/components/TutorGridLogo";
import {
  AuthApiError,
  setActiveTenant,
  type MeResponse,
  type Role,
} from "@/lib/auth";

export interface AuthenticatedHeaderProps {
  me: MeResponse;
  onRefreshMe: () => Promise<void>;
}

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "TG";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0]}${parts[parts.length - 1][0]}`.toUpperCase();
}

function roleBadgeClass(role: Role): string {
  switch (role) {
    case "org_admin":
    case "platform_admin":
      return "border-[#D6C29F]/40 bg-[#D6C29F]/12 text-[#D6C29F]";
    case "teacher":
      return "border-[#8FAF94]/40 bg-[#8FAF94]/12 text-[#8FAF94]";
    default:
      return "border-[#F3F0E8]/18 bg-[#222B24] text-[#E3E1D8]";
  }
}

export function AuthenticatedHeader({
  me,
  onRefreshMe,
}: AuthenticatedHeaderProps) {
  const [switching, setSwitching] = useState(false);
  const [switchError, setSwitchError] = useState<string | null>(null);

  const activeMembership =
    me.memberships.find((m) => m.tenant_id === me.active_tenant_id) ?? null;

  const handleSwitchTenant = async (tenantId: string) => {
    if (!tenantId || tenantId === me.active_tenant_id || switching) return;
    setSwitching(true);
    setSwitchError(null);
    try {
      await setActiveTenant(tenantId);
      await onRefreshMe();
    } catch (err) {
      setSwitchError(
        err instanceof AuthApiError
          ? err.message
          : "Unable to switch teaching space.",
      );
    } finally {
      setSwitching(false);
    }
  };

  return (
    <header className="pointer-events-none fixed inset-x-0 top-0 z-40 w-full pt-4">
      <div className="mx-auto flex max-w-5xl flex-col gap-2 px-4 sm:px-6">
        <div className="flex items-center justify-between gap-3">
          {/* Left Floating Pill: Workspace Switcher + Role Context */}
          <div className="pointer-events-auto flex h-12 items-center gap-2 rounded-full border border-[#F3F0E8]/14 bg-[#181F1A]/86 pr-2.5 pl-2 backdrop-blur-md">
            <TutorGridMark className="h-7 w-7 shrink-0" />

            {me.memberships.length > 0 ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    disabled={switching}
                    className="h-8 gap-2 rounded-full px-2.5 text-xs font-medium text-[#F3F0E8] hover:bg-[#222B24] hover:text-[#F3F0E8]"
                  >
                    <Building2 className="h-3.5 w-3.5 shrink-0 text-[#8FAF94]" />
                    <span className="max-w-[135px] truncate sm:max-w-[200px]">
                      {switching
                        ? "Switching..."
                        : (activeMembership?.tenant_name ?? "Choose Space")}
                    </span>
                    <ChevronDown className="h-3.5 w-3.5 text-[#A6B0A8]" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="start"
                  className="w-64 rounded-xl border border-[#F3F0E8]/14 bg-[#1A221C]/95 p-1.5 text-[#F3F0E8] shadow-none backdrop-blur-md"
                >
                  <DropdownMenuLabel className="px-2.5 py-1.5 text-[11px] font-medium tracking-wider text-[#A6B0A8] uppercase">
                    Teaching Spaces
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator className="my-1 bg-[#F3F0E8]/10" />
                  {me.memberships.map((m) => {
                    const isCurrent = m.tenant_id === me.active_tenant_id;
                    return (
                      <DropdownMenuItem
                        key={m.tenant_id}
                        disabled={switching}
                        onSelect={() => void handleSwitchTenant(m.tenant_id)}
                        className="flex cursor-pointer items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-sm text-[#F3F0E8] focus:bg-[#222B24]"
                      >
                        <div className="min-w-0">
                          <p className="truncate font-medium">{m.tenant_name}</p>
                          <p className="text-[11px] text-[#A6B0A8]">{m.role}</p>
                        </div>
                        {isCurrent && (
                          <Check className="h-4 w-4 shrink-0 text-[#8FAF94]" />
                        )}
                      </DropdownMenuItem>
                    );
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <span className="pr-1.5 text-xs font-medium text-[#D6C29F]">
                Workspace Setup
              </span>
            )}

            {activeMembership && (
              <span
                className={`hidden items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium sm:inline-flex ${roleBadgeClass(activeMembership.role)}`}
              >
                <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                {activeMembership.role}
              </span>
            )}
          </div>

          {/* Right Floating Pill: Account Identity & Sign Out */}
          <div className="pointer-events-auto flex h-12 items-center rounded-full border border-[#F3F0E8]/14 bg-[#181F1A]/86 p-1.5 backdrop-blur-md">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="h-9 gap-2 rounded-full pr-2.5 pl-1 hover:bg-[#222B24]"
                >
                  <Avatar className="h-7 w-7 rounded-full border border-[#D6C29F]/40 bg-[#222B24]">
                    <AvatarFallback className="bg-[#222B24] text-[11px] font-semibold text-[#D6C29F]">
                      {getInitials(me.user.display_name)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="hidden max-w-[120px] truncate text-xs font-medium text-[#F3F0E8] sm:inline">
                    {me.user.display_name}
                  </span>
                  <ChevronDown className="h-3.5 w-3.5 text-[#A6B0A8]" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                className="w-60 rounded-xl border border-[#F3F0E8]/14 bg-[#1A221C]/95 p-1.5 text-[#F3F0E8] shadow-none backdrop-blur-md"
              >
                <DropdownMenuLabel className="px-2.5 py-2">
                  <p className="truncate text-sm font-medium text-[#F3F0E8]">
                    {me.user.display_name}
                  </p>
                  <p className="truncate text-xs font-normal text-[#A6B0A8]">
                    {me.user.email}
                  </p>
                </DropdownMenuLabel>
                <DropdownMenuSeparator className="my-1 bg-[#F3F0E8]/10" />
                <DropdownMenuItem asChild>
                  <Link
                    to="/sign-out"
                    className="flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-2 text-sm text-[#D97373] focus:bg-[#D97373]/12 focus:text-[#D97373]"
                  >
                    <LogOut className="h-4 w-4" />
                    Sign out
                  </Link>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {switchError && (
          <div
            role="alert"
            className="pointer-events-auto self-start rounded-full border border-[#D97373]/40 bg-[#181F1A]/92 px-3.5 py-1.5 text-xs text-[#F4C2C2] backdrop-blur-md"
          >
            {switchError}
          </div>
        )}
      </div>
    </header>
  );
}
