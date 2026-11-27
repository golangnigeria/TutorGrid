import { Link } from "react-router-dom";
import { motion, type Variants } from "motion/react";
import { ArrowRight, BookOpen, PenTool, Video } from "lucide-react";

export interface Hero34Props {
  isAuthenticated: boolean;
}

const titleContainerVariants: Variants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.14, delayChildren: 0.12 },
  },
};

const titleLineVariants: Variants = {
  hidden: { opacity: 0, y: 24 },
  show: {
    opacity: 1,
    y: 0,
    transition: { type: "spring", damping: 28, stiffness: 120 },
  },
};

const bodyContainerVariants: Variants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.1, delayChildren: 0.35 },
  },
};

const bodyItemVariants: Variants = {
  hidden: { opacity: 0, y: 12 },
  show: {
    opacity: 1,
    y: 0,
    transition: { type: "spring", damping: 24, stiffness: 130 },
  },
};

const pillarContainerVariants: Variants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.08, delayChildren: 0.5 },
  },
};

const pillarItemVariants: Variants = {
  hidden: { opacity: 0, y: 10 },
  show: {
    opacity: 1,
    y: 0,
    transition: { type: "spring", damping: 24, stiffness: 140 },
  },
};

const pillars = [
  {
    icon: BookOpen,
    title: "Lesson Studio",
    description:
      "Author versioned lessons and keep reference PDFs organized per teaching space.",
  },
  {
    icon: PenTool,
    title: "Whiteboard Engine",
    description:
      "Teach collaboratively on a dedicated canvas inside calm, distraction-free dark chrome.",
  },
  {
    icon: Video,
    title: "Live Classroom",
    description:
      "Deliver pinned lessons with real-time audio, video, and interactive educational activities.",
  },
];

export function Hero34({ isAuthenticated }: Hero34Props) {
  return (
    <section className="flex min-h-[calc(100vh-14rem)] flex-col justify-between py-6 sm:py-10">
      <div className="max-w-3xl">
        <motion.span
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
          className="inline-flex items-center rounded-full border border-[#D6C29F]/35 bg-[#181F1A]/75 px-3 py-1 text-xs font-medium text-[#D6C29F] backdrop-blur-md"
        >
          TutorGrid Circle · Pre-K through Grade 12
        </motion.span>

        <motion.h1
          variants={titleContainerVariants}
          initial="hidden"
          animate="show"
          className="mt-6 font-display text-4xl leading-[1.08] font-semibold tracking-tight text-[#F3F0E8] sm:text-6xl"
        >
          <motion.span variants={titleLineVariants} className="block">
            Teaching spaces built
          </motion.span>
          <motion.span variants={titleLineVariants} className="block text-[#D6C29F]">
            for real classrooms.
          </motion.span>
        </motion.h1>

        <motion.div
          variants={bodyContainerVariants}
          initial="hidden"
          animate="show"
          className="mt-7 flex flex-col items-start gap-7"
        >
          <motion.p
            variants={bodyItemVariants}
            className="max-w-xl text-base leading-relaxed text-[#E3E1D8] sm:text-lg"
          >
            Bring Lesson Studio, Whiteboard Engine, Educational Activities, and Live Classroom
            together in one calm, tenant-isolated workspace for tutors and schools.
          </motion.p>

          <motion.div
            variants={bodyItemVariants}
            className="flex flex-wrap items-center gap-3.5"
          >
            {isAuthenticated ? (
              <Link
                to="/dashboard"
                className="group inline-flex min-h-12 items-center gap-2 rounded-xl bg-[#8FAF94] px-6 py-3 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
              >
                Open Dashboard
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Link>
            ) : (
              <>
                <Link
                  to="/sign-in"
                  className="group inline-flex min-h-12 items-center gap-2 rounded-xl bg-[#8FAF94] px-6 py-3 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
                >
                  Sign in to Workspace
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
                </Link>
                <Link
                  to="/dashboard"
                  className="inline-flex min-h-12 items-center rounded-xl border border-[#F3F0E8]/20 bg-[#181F1A]/75 px-5 py-3 text-sm font-medium text-[#F3F0E8] backdrop-blur-md transition-colors hover:border-[#D6C29F]/45 hover:bg-[#222B24]/85 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
                >
                  Go to Dashboard
                </Link>
              </>
            )}
          </motion.div>
        </motion.div>
      </div>

      <motion.div
        id="engines"
        variants={pillarContainerVariants}
        initial="hidden"
        animate="show"
        className="mt-14 grid scroll-mt-24 grid-cols-1 gap-4 md:grid-cols-3"
      >
        {pillars.map((pillar) => {
          const Icon = pillar.icon;
          return (
            <motion.div
              key={pillar.title}
              variants={pillarItemVariants}
              className="rounded-2xl border border-[#F3F0E8]/12 bg-[#181F1A]/78 p-5 backdrop-blur-md"
            >
              <Icon className="h-5 w-5 text-[#8FAF94]" aria-hidden="true" />
              <h2 className="mt-3 font-display text-base font-semibold text-[#F3F0E8]">
                {pillar.title}
              </h2>
              <p className="mt-1.5 text-xs leading-relaxed text-[#A6B0A8]">
                {pillar.description}
              </p>
            </motion.div>
          );
        })}
      </motion.div>
    </section>
  );
}
