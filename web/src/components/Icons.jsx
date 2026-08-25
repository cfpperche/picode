import {
  Activity,
  ArrowUp,
  AudioLines,
  Book,
  Bot,
  Check,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Cloud,
  Copy,
  ExternalLink,
  Folder,
  GitBranch,
  Layers,
  List,
  Maximize2,
  MessageSquare,
  Mic,
  Minimize2,
  Monitor,
  Moon,
  Play,
  Plus,
  QrCode,
  SlidersHorizontal,
  Smartphone,
  Square,
  Sun,
  Terminal,
  User,
  Volume2,
  VolumeX,
  X,
} from "lucide-react";

function lucide(Icon, fallback) {
  return function Wrapped({ size = fallback, className, ...p }) {
    return <Icon size={size} className={className} aria-hidden="true" {...p} />;
  };
}

export const IconQR = lucide(QrCode, 15);
export const IconUser = lucide(User, 13);
export const IconChevronUp = lucide(ChevronUp, 14);
export const IconSun = lucide(Sun, 13);
export const IconMonitor = lucide(Monitor, 13);
export const IconPhone = lucide(Smartphone, 13);
export const IconMoon = lucide(Moon, 13);
export const IconChevronRight = lucide(ChevronRight, 13);
export const IconDocs = lucide(Book, 12);
export const IconExternal = lucide(ExternalLink, 13);
export const IconTerminal = lucide(Terminal, 14);
export const IconPlay = lucide(Play, 12);
export const IconStop = lucide(Square, 12);
export const IconProvider = lucide(Cloud, 13);
export const IconModel = lucide(Layers, 13);
export const IconThink = lucide(Activity, 13);
export const IconMode = lucide(SlidersHorizontal, 13);
export const IconCopy = lucide(Copy, 13);
export const IconGit = lucide(GitBranch, 12);
export const IconFolder = lucide(Folder, 13);
export const IconAgent = lucide(Bot, 13);
export const IconSession = lucide(List, 13);
export const IconPlus = lucide(Plus, 13);
export const IconChat = lucide(MessageSquare, 13);
export const IconKind = lucide(MessageSquare, 13);
export const IconSend = lucide(ArrowUp, 14);
export const IconBack = lucide(ChevronLeft, 13);
export const IconX = lucide(X, 16);
export const IconCheck = lucide(Check, 16);
export const IconMic = lucide(Mic, 16);
export const IconWave = lucide(AudioLines, 16);
export const IconSpeaker = lucide(Volume2, 16);
export const IconSpeakerOff = lucide(VolumeX, 16);
export const IconExpand = lucide(Maximize2, 14);
export const IconCollapse = lucide(Minimize2, 14);
export const IconMaximize = lucide(Square, 12);
export const IconRestore = lucide(Copy, 12);
