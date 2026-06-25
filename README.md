# FlipCTL — a UI framework for embedded Linux systems

@startuml
!define DBUS [D-Bus]

package "View Layer" {
  component "HW View\n(hw_viewd)" as HW
  note top of HW : Control\nPanel
  component "Web View\n(web_viewd)" as WEB
  note top of WEB : Web Browser\nInterface
  component "TUI View\n(tui_viewd)" as TUI
note top of TUI : Terminal\nInterface
}

package "Model Layer" {
  component "Model\n(modeld)" as MODEL
}

package "AppWrapper Layer" {
  component "AppWrapper\n(app_wrapperd)" as APP
}

package "System Layer" {
  component "Console\nutilities" as UTILS
}

HW -down-> MODEL : «DBus»
WEB -down-> MODEL : «DBus»
TUI -down-> MODEL : «DBus»
MODEL -down-> APP : «DBus»
APP -down-> UTILS : I/O, Execute


note right of MODEL : (Presenter + Model)
note left of APP : Application Wrapper

@enduml

