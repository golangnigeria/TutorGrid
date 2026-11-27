import { Link, useLocation } from "react-router-dom";
import { motion } from "motion/react";
import { Home, Layers, LayoutDashboard } from "lucide-react";

export interface MobileFooterDockProps {
  isAuthenticated: boolean;
}

interface DockItem {
  key: string;
  label: string;
  to: string;
  icon: typeof Home;
}

const unauthenticatedItems: DockItem[] = [
  { key: "home", label: "Home", to: "/", icon: Home },
  { key: "engines", label: "Engines", to: "/#engines", icon: Layers },
];

// Authenticated dock owns pure route navigation only (Sign out lives exclusively in the top right account pill).
const authenticatedItems: DockItem[] = [
  { key: "home", label: "Home", to: "/", icon: Home },
  { key: "dashboard", label: "Dashboard", to: "/dashboard", icon: LayoutDashboard },
];

export function MobileFooterDock({ isAuthenticated }: MobileFooterDockProps) {
  const location = useLocation();
  const items = isAuthenticated ? authenticatedItems : unauthenticatedItems;

  const isActive = (item: DockItem): boolean => {
    if (item.to === "/#engines") {
      return location.pathname === "/" && location.hash === "#engines";
    }
    if (item.to === "/") {
      return location.pathname === "/" && location.hash !== "#engines";
    }
    return location.pathname === item.to;
  };

  return (
    <nav
      aria-label="Primary bottom navigation"
      className="pointer-events-none fixed inset-x-0 bottom-4 z-40 flex justify-center px-4 pb-[env(safe-area-inset-bottom)]"
    >
      <div className="pointer-events-auto flex items-center gap-1 rounded-full border border-[#F3F0E8]/14 bg-[#181F1A]/88 p-1.5 backdrop-blur-md">
        {items.map((item) => {
          const Icon = item.icon;
          const active = isActive(item);

          return (
            <Link
              key={item.key}
              to={item.to}
              aria-current={active ? "page" : undefined}
              className={`relative flex min-w-[82px] flex-col items-center justify-center gap-0.5 rounded-full px-4 py-2 text-[11px] font-medium transition-colors sm:min-w-[92px] ${
                active
                  ? "text-[#F3F0E8]"
                  : "text-[#A6B0A8] hover:text-[#F3F0E8]"
              }`}
            >
              {active && (
                <motion.span
                  layoutId="mobile-footer-dock-active"
                  transition={{ type: "spring", stiffness: 380, damping: 32 }}
                  className="absolute inset-0 -z-10 rounded-full border border-[#8FAF94]/35 bg-[#8FAF94]/18"
                />
              )}
              <Icon
                className={`h-4 w-4 ${active ? "text-[#8FAF94]" : ""}`}
                aria-hidden="true"
              />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
