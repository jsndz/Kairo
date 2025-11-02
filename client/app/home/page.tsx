"use client";

import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Zap,
  Users,
  Sparkles,
  Code,
  MessageSquare,
  ArrowRight,
  Check,
  Star,
  Rocket,
  Shield,
  Workflow,
} from "lucide-react";

export default function Page() {
  const [scrollY, setScrollY] = useState(0);
  const [activeFeature, setActiveFeature] = useState(0);

  useEffect(() => {
    const handleScroll = () => setScrollY(window.scrollY);
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveFeature((prev) => (prev + 1) % 3);
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  const features = [
    {
      icon: <Users className="w-6 h-6" />,
      title: "Real-Time Collaboration",
      description:
        "Work together seamlessly with live cursors, instant updates, and presence awareness.",
    },
    {
      icon: <Sparkles className="w-6 h-6" />,
      title: "AI-Powered Assistance",
      description:
        "Intelligent copilots help you write, refactor, and document faster than ever.",
    },
    {
      icon: <Code className="w-6 h-6" />,
      title: "Built for Technical Teams",
      description:
        "Rich code blocks, syntax highlighting, and seamless integration with your workflow.",
    },
  ];

  const capabilities = [
    "Rich text editing with advanced formatting",
    "Live collaborative editing with CRDTs",
    "Code syntax highlighting",
    "Real-time presence indicators",
    "Version history tracking",
    "AI-powered suggestions",
  ];

  const stats = [
    { value: "10k+", label: "Active Teams" },
    { value: "99.9%", label: "Uptime" },
    { value: "50ms", label: "Sync Speed" },
    { value: "150+", label: "Countries" },
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-50 text-slate-900 overflow-hidden relative">
      {/* Advanced animated background */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div
          className="absolute w-[600px] h-[600px] bg-black/5 rounded-full blur-3xl animate-pulse"
          style={{
            top: "-200px",
            left: "-200px",
            animationDuration: "8s",
          }}
        />
        <div
          className="absolute w-[500px] h-[500px] bg-black/5 rounded-full blur-3xl animate-pulse"
          style={{
            top: "40%",
            right: "-200px",
            animationDuration: "10s",
            animationDelay: "2s",
          }}
        />
        <div
          className="absolute w-[400px] h-[400px] bg-black/5 rounded-full blur-3xl animate-pulse"
          style={{
            bottom: "-100px",
            left: "30%",
            animationDuration: "12s",
            animationDelay: "4s",
          }}
        />
      </div>

      {/* Grid overlay */}
      <div className="fixed inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAxMCAwIEwgMCAwIDAgMTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzAwMCIgc3Ryb2tlLW9wYWNpdHk9IjAuMDMiIHN0cm9rZS13aWR0aD0iMSIvPjwvcGF0dGVybj48L2RlZnM+PHJlY3Qgd2lkdGg9IjEwMCUiIGhlaWdodD0iMTAwJSIgZmlsbD0idXJsKCNncmlkKSIvPjwvc3ZnPg==')] opacity-40 pointer-events-none" />

      {/* Navigation */}
      <nav className="relative z-50 backdrop-blur-xl bg-white/70 border-b border-slate-200/50 sticky top-0">
        <div className="flex items-center justify-between px-8 py-4 max-w-7xl mx-auto">
          <div className="flex items-center space-x-3">
            <img src="/kairo.svg" alt="Kairo" className="h-10 w-auto" />
          </div>
          <div className="flex items-center space-x-6">
            <Button
              variant="ghost"
              className="text-slate-600 hover:text-slate-900 hover:bg-slate-100"
            >
              Features
            </Button>
            <Button
              variant="ghost"
              className="text-slate-600 hover:text-slate-900 hover:bg-slate-100"
            >
              Pricing
            </Button>
            <Button
              variant="ghost"
              className="text-slate-600 hover:text-slate-900 hover:bg-slate-100"
            >
              Docs
            </Button>
            <Button className="bg-black text-white hover:bg-slate-900">
              Get Started
            </Button>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 pt-24 pb-20">
        <div className="text-center space-y-8">
          <div className="inline-flex items-center space-x-2 px-6 py-2 bg-black/5 rounded-full border border-black/10">
            <Rocket className="w-4 h-4" />
            <span className="text-sm font-medium">Powered by Real-Time AI</span>
          </div>

          <h1 className="text-7xl md:text-8xl font-bold leading-tight tracking-tight">
            <span className="text-slate-900">Documentation</span>
            <br />
            <span className="text-slate-900 italic">Reimagined</span>
          </h1>

          <p className="text-xl md:text-2xl text-slate-600 max-w-3xl mx-auto leading-relaxed">
            The intelligent workspace where technical teams create, collaborate,
            and ship documentation at the speed of thought.
          </p>

          <div className="flex items-center justify-center space-x-4 pt-8">
            <Button
              size="lg"
              className="bg-black text-white hover:bg-slate-900 text-lg px-10 py-6 group transition-all duration-300 hover:scale-105"
            >
              Start Creating Free
              <ArrowRight className="w-5 h-5 ml-2 group-hover:translate-x-1 transition-transform" />
            </Button>
            <Button
              size="lg"
              variant="outline"
              className="border-2 border-slate-300 hover:bg-slate-100 text-lg px-10 py-6 hover:border-slate-400 transition-all duration-300"
            >
              Watch Demo
            </Button>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 pt-16 max-w-4xl mx-auto">
            {stats.map((stat, index) => (
              <div
                key={index}
                className="space-y-2 animate-in fade-in slide-in-from-bottom duration-700"
                style={{ animationDelay: `${index * 100}ms` }}
              >
                <div className="text-4xl font-bold text-slate-900">
                  {stat.value}
                </div>
                <div className="text-sm text-slate-600">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Floating Editor Preview with enhanced styling */}
        <div
          className="mt-24 relative"
          style={{ transform: `translateY(${scrollY * 0.05}px)` }}
        >
          <div className="absolute inset-0 bg-black/5 blur-3xl transform scale-110" />
          <Card className="relative bg-white/80 backdrop-blur-xl border-slate-200/50 shadow-2xl overflow-hidden hover:shadow-3xl transition-all duration-500">
            <CardContent className="p-0">
              <div className="bg-gradient-to-r from-slate-50 to-slate-100 border-b border-slate-200/50 px-6 py-4 flex items-center justify-between">
                <div className="flex items-center space-x-4">
                  <div className="flex space-x-2">
                    <div className="w-3 h-3 rounded-full bg-slate-300 hover:bg-slate-400 transition-colors cursor-pointer" />
                    <div className="w-3 h-3 rounded-full bg-slate-300 hover:bg-slate-400 transition-colors cursor-pointer" />
                    <div className="w-3 h-3 rounded-full bg-slate-300 hover:bg-slate-400 transition-colors cursor-pointer" />
                  </div>
                  <span className="text-sm font-medium text-slate-700">
                    API Documentation.md
                  </span>
                  <div className="flex items-center space-x-1 px-2 py-1 bg-slate-100 text-slate-700 rounded text-xs font-medium">
                    <Star className="w-3 h-3" />
                    Editing
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  <div className="flex -space-x-3">
                    {[
                      { initial: "JD" },
                      { initial: "SM" },
                      { initial: "AK" },
                    ].map((user, i) => (
                      <div
                        key={i}
                        className="w-8 h-8 rounded-full bg-slate-800 border-2 border-white flex items-center justify-center text-xs font-bold text-white shadow-lg hover:scale-110 transition-transform cursor-pointer"
                      >
                        {user.initial}
                      </div>
                    ))}
                  </div>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-xs hover:bg-slate-200/50"
                  >
                    Share
                  </Button>
                </div>
              </div>
              <div className="p-10 space-y-6 bg-white/50">
                <div className="space-y-3">
                  <div className="h-5 bg-gradient-to-r from-slate-200 to-slate-100 rounded-lg w-full animate-pulse" />
                  <div
                    className="h-5 bg-gradient-to-r from-slate-200 to-slate-100 rounded-lg w-5/6 animate-pulse"
                    style={{ animationDelay: "0.1s" }}
                  />
                  <div
                    className="h-5 bg-gradient-to-r from-slate-200 to-slate-100 rounded-lg w-full animate-pulse"
                    style={{ animationDelay: "0.2s" }}
                  />
                </div>
                <div className="space-y-3 pt-4">
                  <div
                    className="h-5 bg-slate-200 rounded-lg w-4/5 animate-pulse"
                    style={{ animationDelay: "0.3s" }}
                  />
                  <div
                    className="h-5 bg-slate-200 rounded-lg w-full animate-pulse"
                    style={{ animationDelay: "0.4s" }}
                  />
                  <div
                    className="h-5 bg-slate-200 rounded-lg w-3/4 animate-pulse"
                    style={{ animationDelay: "0.5s" }}
                  />
                </div>
                <div className="bg-slate-900 rounded-xl p-6 space-y-2 shadow-inner">
                  <div className="h-4 bg-slate-700 rounded w-2/3" />
                  <div className="h-4 bg-slate-700 rounded w-4/5" />
                  <div className="h-4 bg-slate-700 rounded w-1/2" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Features Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32">
        <div className="text-center mb-20">
          <div className="inline-flex items-center space-x-2 px-4 py-1 mb-6 bg-gradient-to-r from-slate-100 to-slate-200 text-slate-700 rounded-full border border-slate-300">
            <span className="text-sm font-medium">Features</span>
          </div>
          <h2 className="text-5xl md:text-6xl font-bold mb-6 text-slate-900">
            Everything You Need
          </h2>
          <p className="text-slate-600 text-xl max-w-2xl mx-auto">
            Powerful features designed for modern technical teams
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <Card
              key={index}
              className={`group bg-white/70 backdrop-blur-sm border-2 transition-all duration-500 hover:scale-105 cursor-pointer relative overflow-hidden ${
                activeFeature === index
                  ? "border-slate-300 shadow-2xl"
                  : "border-slate-200 hover:border-slate-300 hover:shadow-xl"
              }`}
            >
              <CardContent className="relative p-8 space-y-5">
                <div className="w-14 h-14 bg-slate-800 rounded-2xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300">
                  <div className="text-white">{feature.icon}</div>
                </div>
                <h3 className="text-2xl font-bold text-slate-900">
                  {feature.title}
                </h3>
                <p className="text-slate-600 leading-relaxed">
                  {feature.description}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Capabilities Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32">
        <div className="grid md:grid-cols-2 gap-20 items-center">
          <div className="space-y-8">
            <div className="inline-flex items-center space-x-2 px-4 py-1 bg-gradient-to-r from-slate-100 to-slate-200 text-slate-700 rounded-full border border-slate-300">
              <Workflow className="w-3 h-3" />
              <span className="text-sm font-medium">Workflow</span>
            </div>
            <h2 className="text-5xl font-bold leading-tight">
              <span className="text-slate-900">Built for the way</span>
              <br />
              <span className="text-slate-900 italic">
                technical teams work
              </span>
            </h2>
            <p className="text-slate-600 text-lg leading-relaxed">
              Every feature is designed to enhance your documentation workflow,
              from first draft to final review.
            </p>
            <div className="space-y-4 pt-4">
              {capabilities.map((capability, index) => (
                <div
                  key={index}
                  className="flex items-center space-x-4 group animate-in fade-in slide-in-from-left duration-700"
                  style={{ animationDelay: `${index * 100}ms` }}
                >
                  <div className="w-7 h-7 bg-slate-800 rounded-full flex items-center justify-center flex-shrink-0 shadow-lg group-hover:scale-110 transition-transform">
                    <Check className="w-4 h-4 text-white" />
                  </div>
                  <span className="text-slate-700 font-medium group-hover:text-slate-900 transition-colors">
                    {capability}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-0 bg-black/5 blur-3xl" />
            <Card className="relative bg-white/80 backdrop-blur-xl border-slate-200/50 shadow-2xl hover:shadow-3xl transition-all duration-500">
              <CardContent className="p-8 space-y-6">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-slate-800 rounded-xl flex items-center justify-center shadow-lg">
                      <MessageSquare className="w-5 h-5 text-white" />
                    </div>
                    <span className="font-bold text-lg text-slate-900">
                      Live Comments
                    </span>
                  </div>
                  <div className="flex items-center space-x-1 px-2 py-1 bg-slate-100 text-slate-700 rounded text-xs font-medium">
                    <div className="w-2 h-2 bg-slate-500 rounded-full mr-1 animate-pulse" />
                    3 Active
                  </div>
                </div>
                <div className="space-y-5">
                  {[
                    {
                      name: "JD",
                      time: "2 min",
                    },
                    {
                      name: "SM",
                      time: "5 min",
                    },
                    {
                      name: "AK",
                      time: "8 min",
                    },
                  ].map((user, i) => (
                    <div
                      key={i}
                      className="flex items-start space-x-4 group hover:bg-slate-50 p-3 rounded-xl transition-all duration-300"
                    >
                      <div className="w-10 h-10 rounded-xl bg-slate-800 flex-shrink-0 flex items-center justify-center text-white text-sm font-bold shadow-lg group-hover:scale-110 transition-transform">
                        {user.name}
                      </div>
                      <div className="flex-1 space-y-2">
                        <div
                          className="h-3 bg-slate-200 rounded-full"
                          style={{ width: `${70 + i * 10}%` }}
                        />
                        <div
                          className="h-3 bg-slate-200 rounded-full"
                          style={{ width: `${50 + i * 5}%` }}
                        />
                        <div className="text-xs text-slate-400 font-medium mt-2">
                          {user.time} ago
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32">
        <Card className="relative bg-slate-900 border-0 overflow-hidden shadow-2xl group">
          <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAxMCAwIEwgMCAwIDAgMTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0id2hpdGUiIHN0cm9rZS1vcGFjaXR5PSIwLjA1IiBzdHJva2Utd2lkdGg9IjEiLz48L3BhdHRlcm4+PC9kZWZzPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9InVybCgjZ3JpZCkiLz48L3N2Zz4=')] opacity-40" />

          <CardContent className="relative p-20 text-center space-y-8">
            <div className="inline-flex items-center space-x-2 px-6 py-2 bg-white/10 text-white rounded-full border border-white/20">
              <Shield className="w-4 h-4" />
              <span className="text-sm font-medium">Enterprise Ready</span>
            </div>
            <h2 className="text-5xl md:text-6xl font-bold text-white leading-tight">
              Ready to Transform
              <br />
              <span className="text-white">Your Workflow?</span>
            </h2>
            <p className="text-xl text-slate-300 max-w-2xl mx-auto leading-relaxed">
              Join thousands of technical teams who are already creating better
              documentation, faster and smarter.
            </p>
            <div className="flex items-center justify-center space-x-4 pt-8">
              <Button
                size="lg"
                className="bg-white text-slate-900 hover:bg-slate-100 text-lg px-12 py-6 shadow-2xl hover:scale-105 transition-all duration-300"
              >
                Get Started Free
              </Button>
              <Button
                size="lg"
                variant="outline"
                className="border-2 border-white/30 text-white hover:bg-white/10 hover:border-white/50 text-lg px-12 py-6 backdrop-blur-sm transition-all duration-300"
              >
                Schedule Demo
              </Button>
            </div>
            <p className="text-sm text-slate-400 pt-4">
              No credit card required • 14-day free trial • Cancel anytime
            </p>
          </CardContent>
        </Card>
      </section>

      {/* Footer */}
      <footer className="relative z-10 border-t border-slate-200 mt-32 bg-white/50 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-8 py-16">
          <div className="flex flex-col md:flex-row items-center justify-between space-y-6 md:space-y-0">
            <div className="flex items-center space-x-3">
              <img src="/kairo.svg" alt="Kairo" className="h-10 w-auto" />
            </div>
            <div className="flex items-center space-x-8">
              <a
                href="#"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                Privacy
              </a>
              <a
                href="#"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                Terms
              </a>
              <a
                href="#"
                className="text-slate-600 hover:text-slate-900 transition-colors"
              >
                Support
              </a>
            </div>
            <p className="text-slate-500 text-sm">
              © 2025 Kairo. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}
