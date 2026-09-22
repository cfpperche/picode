import {
  Activity,
  ArrowDownToLine,
  ArrowUp,
  ArrowDown,
  AudioLines,
  Bold,
  Book,
  Bot,
  Boxes,
  CircleCheck,
  Code,
  CaseSensitive,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  ClipboardPaste,
  Clock,
  Cloud,
  Copy,
  CornerDownLeft,
  Download,
  Eraser,
  Ellipsis,
  EllipsisVertical,
  ExternalLink,
  File,
  Folder,
  FolderOpen,
  GitPullRequest,
  Link2,
  Lock,
  Maximize,
  Maximize2,
  Minimize2,
  Folders,
  GitBranch,
  Frame,
  FlaskConical,
  HardDrive,
  Home,
  Image,
  Inbox,
  Info,
  LayoutGrid,
  Heading2,
  Italic,
  Layers,
  List,
  ListOrdered,
  MessageSquare,
  Mic,
  Monitor,
  Moon,
  PanelRight,
  PanelRightClose,
  Paperclip,
  PenLine,
  OctagonX,
  Package,
  Pencil,
  Pin,
  Play,
  Keyboard,
  Plug,
  Plus,
  QrCode,
  Quote,
  Regex,
  RotateCw,
  Archive,
  ArchiveRestore,
  Search,
  Star,
  Settings,
  SlidersHorizontal,
  Sparkles,
  Smartphone,
  Square,
  SquareTerminal,
  Sun,
  Terminal,
  TextSelect,
  TriangleAlert,
  Type,
  Trash2,
  Unlink,
  User,
  Volume2,
  VolumeX,
  WholeWord,
  X,
  Globe,
} from "lucide-react";

function lucide(Icon, fallback) {
  return function Wrapped({ size = fallback, className, ...p }) {
    return <Icon size={size} className={className} aria-hidden="true" {...p} />;
  };
}

export const IconQR = lucide(QrCode, 15);
export const IconUser = lucide(User, 13);
export const IconChevronUp = lucide(ChevronUp, 14);
export const IconChevronDown = lucide(ChevronDown, 14);
export const IconSun = lucide(Sun, 13);
export const IconMonitor = lucide(Monitor, 13);
export const IconPhone = lucide(Smartphone, 13);
export const IconMoon = lucide(Moon, 13);
export const IconChevronRight = lucide(ChevronRight, 13);
export const IconChevronLeft = lucide(ChevronLeft, 13);
export const IconDocs = lucide(Book, 12);
export const IconExternal = lucide(ExternalLink, 13);
export const IconTerminal = lucide(Terminal, 14);
// The tmux app's mark: lucide's boxed prompt — the server a terminal lives
// in, which is what this app shows (ADR-0109's icon map owns the key).
export const IconTmux = lucide(SquareTerminal, 14);
export const IconCli = lucide(Boxes, 16);
export const IconPlay = lucide(Play, 12);
export const IconKeyboard = lucide(Keyboard, 18);
export const IconStop = lucide(Square, 12);
export const IconProvider = lucide(Cloud, 13);
export const IconModel = lucide(Layers, 13);
export const IconThink = lucide(Activity, 13);
export const IconMode = lucide(SlidersHorizontal, 13);
export const IconLock = lucide(Lock, 12);
export const IconCopy = lucide(Copy, 13);
export const IconPaste = lucide(ClipboardPaste, 13);
export const IconEnter = lucide(CornerDownLeft, 14);
export const IconReload = lucide(RotateCw, 13);
export const IconDownload = lucide(Download, 13);
export const IconGit = lucide(GitBranch, 12);
export const IconLink = lucide(Link2, 13);
export const IconUnlink = lucide(Unlink, 13);
export const IconPanelRight = lucide(PanelRight, 16);
export const IconPanelRightClose = lucide(PanelRightClose, 16);
export const IconRemote = lucide(Cloud, 10);
export const IconFolder = lucide(Folder, 13);
export const IconFolders = lucide(Folders, 13);
export const IconFolderOpen = lucide(FolderOpen, 13);
export const IconPullRequest = lucide(GitPullRequest, 13);
export const IconMoveUp = lucide(ArrowUp, 13);
export const IconMoveDown = lucide(ArrowDown, 13);
export const IconGrid = lucide(LayoutGrid, 13);
// The Canvas app's tile (ADR-0109 icon map). A 3×3 grid drew the engine
// ADR-0118 removed, so the glyph is a frame with its guides: a bounded plane
// with things placed on it, which is what the app now is.
export const IconCanvas = lucide(Frame, 13);
export const IconFlask = lucide(FlaskConical, 13);
export const IconInbox = lucide(Inbox, 13);
export const IconClock = lucide(Clock, 13);
export const IconTrash = lucide(Trash2, 13);
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
// Fit-to-view: four corner brackets, the glyph React Flow's own Controls draw
// for it (owner, 2026-09-12). Maximize2's diagonal arrows read as "make this
// bigger"; the brackets read as "frame everything", which is what Fit does.
export const IconFit = lucide(Maximize, 14);
export const IconCollapse = lucide(Minimize2, 14);
export const IconMaximize = lucide(Square, 12);
export const IconRestore = lucide(Copy, 12);
export const IconPin = lucide(Pin, 13);
export const IconSettings = lucide(Settings, 14);
export const IconSparkles = lucide(Sparkles, 14);
export const IconMcp = lucide(Plug, 14);
export const IconPackage = lucide(Package, 14);
export const IconClip = lucide(Paperclip, 13);
export const IconImage = lucide(Image, 13);
export const IconSketch = lucide(PenLine, 13);
export const IconPencil = lucide(Pencil, 12);
export const IconSelectAll = lucide(TextSelect, 13);
export const IconScrollEnd = lucide(ArrowDownToLine, 13);
export const IconTextSize = lucide(Type, 13);
export const IconClear = lucide(Eraser, 13);
export const IconSearch = lucide(Search, 13);
export const IconStar = lucide(Star, 13);
export const IconArchive = lucide(Archive, 13);
export const IconArchiveRestore = lucide(ArchiveRestore, 13);
export const IconCase = lucide(CaseSensitive, 15);
export const IconRegex = lucide(Regex, 15);
export const IconWholeWord = lucide(WholeWord, 15);
export const IconBold = lucide(Bold, 14);
export const IconItalic = lucide(Italic, 14);
export const IconHeading = lucide(Heading2, 14);
export const IconList = lucide(List, 14);
export const IconListOl = lucide(ListOrdered, 14);
export const IconCode = lucide(Code, 14);
export const IconQuote = lucide(Quote, 14);

// Notice levels (docs/benchmarks/2026-09-07-superset-notifications.md):
// the glyph is what a one-line notice has instead of a coloured card.
export const IconOk = lucide(CircleCheck, 14);
export const IconInfo = lucide(Info, 14);
export const IconWarn = lucide(TriangleAlert, 14);
export const IconError = lucide(OctagonX, 14);

export const IconGlobe = lucide(Globe, 13);

// The brand mark — the exact geometry of /favicon.svg (the official pi
// glyph), so the shell header wears what the browser tab and the
// installed app show. Glyph only: the plate is the chip's own background.
export function IconBrandMark({ size = 10 }) {
  return (
    <svg width={size} height={size} viewBox="165.29 165.29 469.43 469.43" aria-hidden="true" focusable="false">
      <path fill="#fff" fillRule="evenodd" d="M165.29 165.29H517.36V400H400V517.36H282.65V634.72H165.29ZM282.65 282.65V400H400V282.65Z" />
      <path fill="#fff" d="M517.36 400H634.72V634.72H517.36Z" />
    </svg>
  );
}
