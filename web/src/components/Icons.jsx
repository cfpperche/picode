import {
  Activity,
  ArrowUp,
  AudioLines,
  Bold,
  Book,
  Bot,
  Code,
  Check,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Clock,
  Cloud,
  Copy,
  Download,
  Ellipsis,
  EllipsisVertical,
  ExternalLink,
  File,
  Folder,
  Lock,
  Folders,
  GitBranch,
  FlaskConical,
  HardDrive,
  Home,
  Inbox,
  LayoutGrid,
  Heading2,
  Italic,
  Layers,
  List,
  ListOrdered,
  Maximize2,
  MessageSquare,
  Mic,
  Minimize2,
  Monitor,
  Moon,
  Paperclip,
  PenLine,
  Package,
  Pencil,
  Pin,
  Play,
  Plug,
  Plus,
  QrCode,
  Quote,
  Settings,
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
export const IconChevronLeft = lucide(ChevronLeft, 13);
export const IconDocs = lucide(Book, 12);
export const IconExternal = lucide(ExternalLink, 13);
export const IconTerminal = lucide(Terminal, 14);
export const IconPlay = lucide(Play, 12);
export const IconStop = lucide(Square, 12);
export const IconProvider = lucide(Cloud, 13);
export const IconModel = lucide(Layers, 13);
export const IconThink = lucide(Activity, 13);
export const IconMode = lucide(SlidersHorizontal, 13);
export const IconLock = lucide(Lock, 12);
export const IconCopy = lucide(Copy, 13);
export const IconDownload = lucide(Download, 13);
export const IconGit = lucide(GitBranch, 12);
export const IconRemote = lucide(Cloud, 10);
export const IconFolder = lucide(Folder, 13);
export const IconFolders = lucide(Folders, 13);
export const IconGrid = lucide(LayoutGrid, 13);
export const IconFlask = lucide(FlaskConical, 13);
export const IconInbox = lucide(Inbox, 13);
export const IconClock = lucide(Clock, 13);
export const IconFile = lucide(File, 13);
export const IconMore = lucide(EllipsisVertical, 14);
export const IconEllipsis = lucide(Ellipsis, 14);
export const IconHome = lucide(Home, 13);
export const IconDrive = lucide(HardDrive, 13);
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
export const IconPin = lucide(Pin, 13);
export const IconSettings = lucide(Settings, 14);
export const IconMcp = lucide(Plug, 14);
export const IconPackage = lucide(Package, 14);
export const IconClip = lucide(Paperclip, 13);
export const IconSketch = lucide(PenLine, 13);
export const IconPencil = lucide(Pencil, 12);
export const IconBold = lucide(Bold, 14);
export const IconItalic = lucide(Italic, 14);
export const IconHeading = lucide(Heading2, 14);
export const IconList = lucide(List, 14);
export const IconListOl = lucide(ListOrdered, 14);
export const IconCode = lucide(Code, 14);
export const IconQuote = lucide(Quote, 14);
