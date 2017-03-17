use std::str::FromStr;

#[allow(non_camel_case_types)]
#[derive(Copy,Clone)]
pub enum Facility {
  LOG_KERN     = 0  << 3,
  LOG_USER     = 1  << 3,
  LOG_MAIL     = 2  << 3,
  LOG_DAEMON   = 3  << 3,
  LOG_AUTH     = 4  << 3,
  LOG_SYSLOG   = 5  << 3,
  LOG_LPR      = 6  << 3,
  LOG_NEWS     = 7  << 3,
  LOG_UUCP     = 8  << 3,
  LOG_CRON     = 9  << 3,
  LOG_AUTHPRIV = 10 << 3,
  LOG_FTP      = 11 << 3,
  LOG_LOCAL0   = 16 << 3,
  LOG_LOCAL1   = 17 << 3,
  LOG_LOCAL2   = 18 << 3,
  LOG_LOCAL3   = 19 << 3,
  LOG_LOCAL4   = 20 << 3,
  LOG_LOCAL5   = 21 << 3,
  LOG_LOCAL6   = 22 << 3,
  LOG_LOCAL7   = 23 << 3
}

impl FromStr for Facility {
    type Err = ();
    fn from_str(s: &str) -> Result<Facility, ()> {
        let result = match &s.to_lowercase()[..] {
            "log_kern"    | "kern"     => Facility::LOG_KERN,
            "log_user"    | "user"     => Facility::LOG_USER,
            "log_mail"    | "mail"     => Facility::LOG_MAIL,
            "log_daemon"  | "daemon"   => Facility::LOG_DAEMON,
            "log_auth"    | "auth"     => Facility::LOG_AUTH,
            "log_syslog"  | "syslog"   => Facility::LOG_SYSLOG,
            "log_lpr"     | "lpr"      => Facility::LOG_LPR,
            "log_news"    | "news"     => Facility::LOG_NEWS,
            "log_uucp"    | "uucp"     => Facility::LOG_UUCP,
            "log_cron"    | "cron"     => Facility::LOG_CRON,
            "log_authpriv"| "authpriv" => Facility::LOG_AUTHPRIV,
            "log_ftp"     | "ftp"      => Facility::LOG_FTP,
            "log_local0"  | "local0"   => Facility::LOG_LOCAL0,
            "log_local1"  | "local1"   => Facility::LOG_LOCAL1,
            "log_local2"  | "local2"   => Facility::LOG_LOCAL2,
            "log_local3"  | "local3"   => Facility::LOG_LOCAL3,
            "log_local4"  | "local4"   => Facility::LOG_LOCAL4,
            "log_local5"  | "local5"   => Facility::LOG_LOCAL5,
            "log_local6"  | "local6"   => Facility::LOG_LOCAL6,
            "log_local7"  | "local7"   => Facility::LOG_LOCAL7,
            _ => return Err(())
        };
        Ok(result)
    }
}
