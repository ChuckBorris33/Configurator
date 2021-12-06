import store from './store'
import util from 'util';

export enum LogLevel {
  Trace,
  Debug,
  Info,
  Warning,
  Error,
}

export const LevelNames = {
  [LogLevel.Trace]: "trace",
  [LogLevel.Debug]: "debug",
  [LogLevel.Info]: "info",
  [LogLevel.Warning]: "warning",
  [LogLevel.Error]: "error",
}

export class Log {

  public static trace(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Trace, prefix, ...data);
  }

  public static debug(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Debug, prefix, ...data);
  }

  public static info(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Info, prefix, ...data);
  }

  public static warn(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Warning, prefix, ...data);
  }

  public static warning(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Warning, prefix, ...data);
  }

  public static error(prefix: string, ...data: any[]) {
    Log.log(LogLevel.Error, prefix, ...data);
  }

  public static log(level: LogLevel, prefix?: string, ...data: any[]) {
    let str = "[" + LevelNames[level] + "]";
    if (prefix && prefix.length) {
      str += "[" + prefix + "]";
    }
    if (typeof data[0] == "string") {
      str += data.shift();
    }
    str += " ";
    switch (level) {
      case LogLevel.Debug:
      case LogLevel.Info:
        console.log(str, ...data);
        store.commit('append_log', util.format(str, ...data));
        break;
      case LogLevel.Warning:
        console.warn(str, ...data);
        store.commit('append_log', util.format(str, ...data));
        break;
      case LogLevel.Error:
        console.error(str, ...data);
        store.commit('append_log', util.format(str, ...data));
        break;
      default:
        break;
    }
  }
}